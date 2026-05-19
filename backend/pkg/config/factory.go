package config

import "os"

func NewConfig() Config {
	return Config{
		Host:        env("CONFIG_MAN_HOST", "0.0.0.0"),
		Port:        env("CONFIG_MAN_PORT", "3000"),
		DatabaseURL: env("DATABASE_URL", ""),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
