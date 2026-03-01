package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}
