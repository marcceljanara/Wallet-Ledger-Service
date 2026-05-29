package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort    string        `mapstructure:"SERVER_PORT"`
	DatabaseURL   string        `mapstructure:"DATABASE_URL"`
	RedisURL      string        `mapstructure:"REDIS_URL"`
	RabbitMQURL   string        `mapstructure:"RABBITMQ_URL"`
	JWTSecret     string        `mapstructure:"JWT_SECRET"`
	JWTExpiration time.Duration `mapstructure:"JWT_EXPIRATION"`
}

func Load() (*Config, error) {
	// Load .env file (ignore error if not present — env vars may be set directly)
	_ = godotenv.Load()

	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("JWT_EXPIRATION", "24h")

	cfg := &Config{
		ServerPort:  viper.GetString("SERVER_PORT"),
		DatabaseURL: viper.GetString("DATABASE_URL"),
		RedisURL:    viper.GetString("REDIS_URL"),
		RabbitMQURL: viper.GetString("RABBITMQ_URL"),
		JWTSecret:   viper.GetString("JWT_SECRET"),
	}

	// Parse duration
	expStr := viper.GetString("JWT_EXPIRATION")
	dur, err := time.ParseDuration(expStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRATION: %w", err)
	}
	cfg.JWTExpiration = dur

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	if cfg.RabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}

	return cfg, nil
}
