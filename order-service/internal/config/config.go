package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPC  GRPCConfig
	Mongo MongoConfig
	NATS  NATSConfig
}

type GRPCConfig struct {
	Port string
}

type MongoConfig struct {
	URI string
	DB  string
}

type NATSConfig struct {
	URL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		GRPC: GRPCConfig{
			Port: getEnv("GRPC_PORT", "50051"),
		},
		Mongo: MongoConfig{
			URI: getEnv("MONGO_URI", "mongodb://localhost:27017"),
			DB:  getEnv("MONGO_DB", "orderdb"),
		},
		NATS: NATSConfig{
			URL: getEnv("NATS_URL", "nats://localhost:4222"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.GRPC.Port == "" {
		return fmt.Errorf("GRPC_PORT is required")
	}
	if c.Mongo.URI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}
	if c.Mongo.DB == "" {
		return fmt.Errorf("MONGO_DB is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
