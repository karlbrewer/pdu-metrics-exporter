package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

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

	var response DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Failed to decode agent request, %s", err)
		return nil, err
	}

	return &response, nil
}
