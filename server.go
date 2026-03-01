package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type TokenCacheEntry struct {
	Token    string
	Creation time.Time
}

type Server struct {
	Config *Config

	CacheMutex sync.Mutex
	TokenCache map[string]*TokenCacheEntry
}

func NewServer(config *Config) *Server {
	server := &Server{
		Config:     config,
		TokenCache: map[string]*TokenCacheEntry{},
	}

	return server
}

func (server *Server) CacheToken(key, token string) error {
	server.CacheMutex.Lock()
	defer server.CacheMutex.Unlock()

	server.TokenCache[key] = &TokenCacheEntry{
		Token:    token,
		Creation: time.Now(),
	}

	return nil
}

func (server *Server) GetCachedToken(key string) (string, bool) {
	server.CacheMutex.Lock()
	defer server.CacheMutex.Unlock()

	entry, ok := server.TokenCache[key]
	if !ok {
		log.Printf("No cache entry for %s", key)
		return "", false
	}

	if time.Since(entry.Creation) > server.Config.TokenCacheLifetime {
		log.Printf("Expired cache entry for %s", key)
		delete(server.TokenCache, key)
		return "", false
	}

	log.Printf("Valid cache entry for %s", key)
	return entry.Token, true
}

func parseFloatString(input string) float64 {
	val, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
	if err != nil {
		return 0
	}
	return val
}

func AddGuage(registry *prometheus.Registry, name, help, value string) error {
	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
	err := registry.Register(g)
	if err != nil {
		return err
	}
	g.Set(parseFloatString(value))

	return nil
}

func (server *Server) BuildMetrics(registry *prometheus.Registry, system *DashSystemStatus, power *DashPowerConsumption) error {
	if err := AddGuage(registry, "total_max_watts", "Total max watts", power.TotalMaxWatts); err != nil {
		return err
	}

	if err := AddGuage(registry, "total_used_watts", "Total used watts", power.TotalUsedWatts); err != nil {
		return err
	}

	if err := AddGuage(registry, "total_available_watts", "Total available watts", power.TotalAvailableWatts); err != nil {
		return err
	}

	outlets := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "outlet_watts",
			Help: "Watts for each outlet (label: outlet)",
		},
		[]string{"outlet"},
	)
	registry.Register(outlets)
	outlets.WithLabelValues("1").Set(parseFloatString(power.Out1Watts))
	outlets.WithLabelValues("2").Set(parseFloatString(power.Out2Watts))
	outlets.WithLabelValues("3").Set(parseFloatString(power.Out3Watts))
	outlets.WithLabelValues("4").Set(parseFloatString(power.Out4Watts))
	outlets.WithLabelValues("5").Set(parseFloatString(power.Out5Watts))
	outlets.WithLabelValues("6").Set(parseFloatString(power.Out6Watts))

	if err := AddGuage(registry, "voltage", "Input voltage", system.Voltage); err != nil {
		return err
	}

	if err := AddGuage(registry, "total_current", "Total current draw", system.TotalCurrent); err != nil {
		return err
	}

	if err := AddGuage(registry, "frequency", "Frequency in Hz", system.Frequency); err != nil {
		return err
	}

	if err := AddGuage(registry, "temperature", "System temperature", system.Temperature[:len(system.Temperature)-3]); err != nil {
		return err
	}

	return nil
}

func (server *Server) ProbeHandler(w http.ResponseWriter, r *http.Request) {
	username := server.Config.DefaultUsername
	password := server.Config.DefaultPassword
	insecureSkipVerify := server.Config.DefaultInsecureSkipVerify

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Target parameter is missing", http.StatusBadRequest)
		return
	}

	if targetConfig, ok := server.Config.Pdus[target]; ok {
		if targetConfig.Username != "" {
			username = targetConfig.Username
		}
		if targetConfig.Password != "" {
			password = targetConfig.Password
		}
		insecureSkipVerify = targetConfig.InsecureSkipVerify
	}

	usernameParam := r.URL.Query().Get("username")
	if usernameParam != "" {
		username = usernameParam
	}

	passwordParam := r.URL.Query().Get("password")
	if passwordParam != "" {
		password = passwordParam
	}

	token, ok := server.GetCachedToken(target)
	if !ok {
		loginResponse, err := Login(target, username, password, insecureSkipVerify)
		if err != nil {
			http.Error(w, "Failed to login", http.StatusBadRequest)
			return
		}
		server.CacheToken(target, loginResponse.Token)
		token = loginResponse.Token
	}

	response, err := GetDashboardAgent(target, token, insecureSkipVerify)
	if err != nil {
		http.Error(w, "Failed to login", http.StatusBadRequest)
		return
	}

	reg := prometheus.NewRegistry()

	err = server.BuildMetrics(reg, &response.DashSystemStatus, &response.DashPowerConsumption)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to build metrics", http.StatusBadRequest)
		return
	}

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		Timeout: 5 * time.Second,
	})
	h.ServeHTTP(w, r)
}
