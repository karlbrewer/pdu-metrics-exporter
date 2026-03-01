package main

import (
	"log"
	"net/http"
	"strconv"
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
		log.Println("No cache entry")
		return "", false
	}

	if time.Since(entry.Creation) > server.Config.TokenCacheLifetime {
		log.Println("Expired cache entry")
		delete(server.TokenCache, key)
		return "", false
	}

	log.Println("Valid cache entry")
	return entry.Token, true
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

	log.Println(response)

	reg := prometheus.NewRegistry()

	g := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "total_used_watts",
		Help: "Total used watts",
	})
	err = reg.Register(g)
	if err != nil {
		http.Error(w, "Failed to register gauge", http.StatusBadRequest)
		return
	}

	val, err := strconv.ParseFloat(response.DashPowerConsumption.TotalUsedWatts, 64)
	if err != nil {
		http.Error(w, "Failed to parse float", http.StatusBadRequest)
		return
	}
	log.Printf("Val: %f, %s %v", val, response.DashPowerConsumption.TotalUsedWatts, response.DashPowerConsumption)

	g.Set(val)

	h := promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		Timeout: 5 * time.Second,
	})
	h.ServeHTTP(w, r)
}
