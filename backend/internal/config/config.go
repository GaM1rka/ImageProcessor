package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort     string
	StoragePath  string
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string
}

func Load() Config {
	return Config{
		HTTPPort:     env("HTTP_PORT", "8080"),
		StoragePath:  env("STORAGE_PATH", "./data"),
		KafkaBrokers: strings.Split(env("KAFKA_BROKERS", "localhost:9092"), ","),
		KafkaTopic:   env("KAFKA_TOPIC", "image-processing"),
		KafkaGroupID: env("KAFKA_GROUP_ID", "image-processor-worker"),
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
