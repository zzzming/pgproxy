package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	TargetHost         string
	TargetPort         string
	ConnectionPoolSize int
}

func envBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}

func envInt(name string, defaultValue int) (int, error) {
	if strV, exists := os.LookupEnv(name); exists {
		v, err := strconv.Atoi(strV)
		if err != nil {
			return defaultValue, fmt.Errorf("env %s err %v", name, err)
		}
		return v, nil
	}
	return defaultValue, nil
}

func envString(name, defaultValue string) string {
	if strV, exists := os.LookupEnv(name); exists {
		return strV
	}
	return defaultValue
}

func NewConfig() (*Config, error) {
	cfg := Config{}
	var err error

	cfg.TargetPort = envString("POSTGRES_PORT", "5432")
	cfg.ConnectionPoolSize, err = envInt("CONN_POOL_SIZE", 100)
	if err != nil {
		return nil, err
	}
	if strV, exists := os.LookupEnv("TARGET_HOST"); exists {
		cfg.TargetHost = strV
	} else {
		return nil, fmt.Errorf("target postgres must be configured")
	}
	return &cfg, nil
}
