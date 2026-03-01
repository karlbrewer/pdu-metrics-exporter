package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.yaml.in/yaml/v4"
)

type PDU struct {
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
	Username           string `yaml:"username"`
	Password           string `yaml:"password"`
}

type Config struct {
	Host                      string         `yaml:"host"`
	Port                      int            `yaml:"port"`
	RequestTimeout            time.Duration  `yaml:"request_timeout"`
	DefaultUsername           string         `yaml:"default_username"`
	DefaultPassword           string         `yaml:"default_password"`
	DefaultInsecureSkipVerify bool           `yaml:"default_insecure_skip_verify"`
	Pdus                      map[string]PDU `yaml:"pdus"`
}

var DefaultConfig *Config = &Config{
	Host:            "0.0.0.0",
	Port:            8080,
	DefaultUsername: "admin",
	DefaultPassword: "foo",
}

func loadConfig(path string) (*Config, error) {
	config := DefaultConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

type Server struct {
	Config *Config
}

func NewServer(config *Config) *Server {
	server := &Server{
		Config: config,
	}

	return server
}

func main() {
	configPath := flag.String("config", "config.yaml", "path to the YAML configuration")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal("Failed to parse configuration. %s", err)
	}

	server := NewServer(config)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/probe", server.ProbeHandler)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	log.Printf("Exporter running on %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type DashSwitchInfo struct {
	Uptime    string `json:"uptime"`
	FWVersion string `json:"fw_version"`
}

type DashSystemStatus struct {
	Voltage      string `json:"voltage"`
	TotalCurrent string `json:"total_current"`
	Frequency    string `json:"frequency"`
	Temperature  string `json:"temperature"`
}

type DashAdminInfo struct {
	SystemName     string `json:"system_name"`
	SystemLocation string `json:"system_location"`
	SystemContact  string `json:"system_contact"`
	SerialNo       string `json:"serial_no"`
}

type DashEtherInfo struct {
	MACAddr        string `json:"mac_addr"`
	IPv4Addr       string `json:"ipv4_addr"`
	IPv4Mask       string `json:"ipv4_mask"`
	IPv4Gate       string `json:"ipv4_gate"`
	IPv4DHCP       bool   `json:"ipv4_dhcp"`
	IPv6DHCP       bool   `json:"ipv6_dhcp"`
	IPv6GlobalAddr string `json:"ipv6_global_addr"`
	IPv6GateAddr   string `json:"ipv6_gate_addr"`
	IPv6LocalAddr  string `json:"ipv6_local_addr"`
}

type DashPowerConsumption struct {
	TotalMaxWatts       string `json:"total_max_watts"`
	TotalUsedWatts      string `json:"total_used_watts"`
	TotalAvailableWatts string `json:"total_available_watts"`
	Out1Watts           string `json:"out1_watts"`
	Out2Watts           string `json:"out2_watts"`
	Out3Watts           string `json:"out3_watts"`
	Out4Watts           string `json:"out4_watts"`
	Out5Watts           string `json:"out5_watts"`
	Out6Watts           string `json:"out6_watts"`
}

type DeviceResponse struct {
	DashSwitchInfo       DashSwitchInfo       `json:"dashSwitchInfo"`
	DashSystemStatus     DashSystemStatus     `json:"dashSystemStatus"`
	DashAdminInfo        DashAdminInfo        `json:"dashAdminInfo"`
	DashEtherInfo        DashEtherInfo        `json:"dashEtherInfo"`
	DashPowerConsumption DashPowerConsumption `json:"dashPowerConsumption"`
	ErrCode              int                  `json:"errCode"`
	Message              string               `json:"message"`
	Status               string               `json:"status"`
	Msg                  string               `json:"msg"`
}

func GetDashboardAgent(host, token string, insecureSkipVerify bool) (*DeviceResponse, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
		Timeout: 15 * time.Second,
	}

	url := fmt.Sprintf("https://%s/api/dashboard/agent?", host)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("Failed to build request, %s", err)
		return nil, err
	}

	bearer := fmt.Sprintf("bearer %s", token)
	req.Header.Set("Authorization", bearer)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send agent request, %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Got error code: %d", resp.StatusCode)
		return nil, fmt.Errorf("failed login request. StatusCode: %d", resp.StatusCode)
	}

	// read body for debug, but allow reuse
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read response body for debug: %v", err)
	} else {
		log.Printf("response body: %s", string(buf))
		// restore body so json.Decoder can read it later
		resp.Body = io.NopCloser(bytes.NewReader(buf))
	}

	var response DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode agent request, %s", err)
		return nil, err
	}

	return &response, nil
}

type LoginRequest struct {
	Credentials LoginCredentials `json:"userLogin"`
}

type LoginCredentials struct {
	Username string `json:"usr"`
	Password string `json:"pwd"`
}

type LoginResponse struct {
	Token      string `json:"token"`
	FirstLogin bool   `json:"first_login"`
	Timeout    int    `json:"timeout"`
	ErrCode    int    `json:"errCode"`
	Message    string `json:"message"`
	Status     string `json:"status"`
	Msg        string `json:"msg"`
}

func Login(host, username, password string, insecureSkipVerify bool) (*LoginResponse, error) {
	log.Printf("Login to %s with user: %s", host, username)
	payload, err := json.Marshal(
		LoginRequest{
			Credentials: LoginCredentials{
				Username: username,
				Password: password,
			},
		},
	)
	if err != nil {
		log.Println("Failed to marshal json, %s", err)
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
		Timeout: 15 * time.Second,
	}

	url := fmt.Sprintf("https://%s/api/sys/login", host)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("Failed to build request, %s", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send login request, %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Got error code: %d", resp.StatusCode)
		return nil, fmt.Errorf("failed login request. StatusCode: %d", resp.StatusCode)
	}

	var response LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode login request, %s", err)
		return nil, err
	}

	return &response, nil
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

	loginResponse, err := Login(target, username, password, insecureSkipVerify)
	if err != nil {
		http.Error(w, "Failed to login", http.StatusBadRequest)
		return
	}

	log.Println(loginResponse)

	response, err := GetDashboardAgent(target, loginResponse.Token, insecureSkipVerify)
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
