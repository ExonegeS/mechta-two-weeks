package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type (
	Config struct {
		Server          Server
		WorkerConfig    WorkerConfig
		ExternalService ExternalService
	}

	Server struct {
		Address string
		Port    string
	}

	WorkerConfig struct {
		MaxWorkers int64
		MaxJobs    int64
	}

	ExternalService struct {
		Host    string
		Port    string
		Timeout time.Duration
	}
)

func NewConfig() *Config {
	return &Config{
		Server{
			Address: getEnvStr("ADDRESS", ""),
			Port:    getEnvStr("PORT", "8080"),
		},
		WorkerConfig{
			MaxWorkers: getEnvInt64("SESSION_TOKEN_LENGTH", 10),
			MaxJobs:    getEnvInt64("SESSION_TOKEN_LENGTH", 1000),
		},
		ExternalService{
			Host:    getEnvStr("SERVER_HOST", "http://localhost"),
			Port:    getEnvStr("SERVER_PORT", "6969"),
			Timeout: time.Duration(getEnvInt64("SERVER_TIMEOUT", 60*2)) * time.Second,
		},
	}
}

func (s *ExternalService) MakeAddressString() string {
	return fmt.Sprintf(
		"%s:%s",
		s.Host,
		s.Port,
	)
}

func getEnvStr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fallback
		}

		return i
	}

	return fallback
}
