package main

import (
	"go.yaml.in/yaml/v4"
	"os"
	"time"
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
	TokenCacheLifetime        time.Duration  `yaml:"token_cache_lifetime"`
	DefaultUsername           string         `yaml:"default_username"`
	DefaultPassword           string         `yaml:"default_password"`
	DefaultInsecureSkipVerify bool           `yaml:"default_insecure_skip_verify"`
	Pdus                      map[string]PDU `yaml:"pdus"`
}

var DefaultConfig *Config = &Config{
	Host:               "0.0.0.0",
	Port:               8080,
	TokenCacheLifetime: time.Hour,
	DefaultUsername:    "admin",
	DefaultPassword:    "foo",
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
