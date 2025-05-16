package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type (
	Config struct {
		Server          Server
		WorkerConfig    WorkerConfig
		ExternalService ExternalService
	}

	Server struct {
		Address  string
		Port     string
		GRPCPort string
	}

	WorkerConfig struct {
		MaxWorkers int64
		BatchSize  int64
	}

	ExternalService struct {
		URI       string
		Timeout   time.Duration
		SecretKey string
	}
)

func NewConfig(filename string) *Config {
	if filename != "" {
		if err := godotenv.Load(filename); err != nil {
			panic(fmt.Errorf("error loading '%s' (.env) file: %w", filename, err))
		}
	}
	return &Config{
		Server{
			Address:  getEnvStr("ADDRESS", ""),
			Port:     getEnvStr("PORT", "8080"),
			GRPCPort: getEnvStr("GRPC_PORT", "50051"),
		},
		WorkerConfig{
			MaxWorkers: getEnvInt64("MAX_WORKERS", 3),
			BatchSize:  getEnvInt64("BATCH_SIZE", 3000),
		},
		ExternalService{
			URI:       mustEnvStr("URI"),
			Timeout:   time.Duration(getEnvInt64("SERVER_TIMEOUT", 120)) * time.Second,
			SecretKey: mustEnvStr("SECRET_KEY"),
		},
	}
}

func getEnvStr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func mustEnvStr(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		panic(fmt.Errorf("environment variable '%s' not set", key))
	}
	return value
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
