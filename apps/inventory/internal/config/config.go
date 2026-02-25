package config

import (
	"fmt"
	"os"
)

type Config struct {
	GRPCPort     string
	DatabaseDSN  string
	KafkaBroker  string
	KafkaGroupID string
}

func Load() *Config {
	return &Config{
		GRPCPort: envOrDefault("GRPC_PORT", "50052"),
		DatabaseDSN: envOrDefault("DATABASE_DSN",
			fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
				envOrDefault("PG_USER", "inventory_user"),
				envOrDefault("PG_PASSWORD", "inventory_secret"),
				envOrDefault("PG_HOST", "localhost"),
				envOrDefault("PG_PORT", "5433"),
				envOrDefault("PG_DATABASE", "inventory"),
			)),
		KafkaBroker:  envOrDefault("KAFKA_BROKER", "localhost:9092"),
		KafkaGroupID: envOrDefault("KAFKA_GROUP_ID", "inventory-service"),
	}
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
