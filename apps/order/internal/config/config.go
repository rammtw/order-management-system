package config

import (
	"fmt"
	"os"
)

type Config struct {
	GRPCPort    string
	DatabaseDSN string
	KafkaBroker string
}

func Load() *Config {
	return &Config{
		GRPCPort: envOrDefault("GRPC_PORT", "50051"),
		DatabaseDSN: envOrDefault("DATABASE_DSN",
			fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				envOrDefault("PG_USER", "postgres"),
				envOrDefault("PG_PASSWORD", "postgres"),
				envOrDefault("PG_HOST", "localhost"),
				envOrDefault("PG_PORT", "5432"),
				envOrDefault("PG_DATABASE", "orders"),
			)),
		KafkaBroker: envOrDefault("KAFKA_BROKER", "localhost:9092"),
	}
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
