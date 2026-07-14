package config

import (
	"fmt"
	"os"
	"time"
)

const (
	defaultServerPort      = "8080"
	defaultTokenTTL        = 24 * time.Hour
	defaultRefreshTokenTTL = 7 * 24 * time.Hour
)

type Config struct {
	ServerPort      string
	DBAddress       string
	JWTSecret       string
	TokenTTL        time.Duration
	RefreshTokenTTL time.Duration
}

func Load() (Config, error) {
	config := Config{
		ServerPort:      defaultServerPort,
		DBAddress:       os.Getenv("DB_ADDRESS"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		TokenTTL:        defaultTokenTTL,
		RefreshTokenTTL: defaultRefreshTokenTTL,
	}

	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		config.ServerPort = serverPort
	}

	if tokenTTL := os.Getenv("TOKEN_TTL"); tokenTTL != "" {
		parsedTTL, err := time.ParseDuration(tokenTTL)
		if err != nil {
			return Config{}, fmt.Errorf("parse TOKEN_TTL: %w", err)
		}

		config.TokenTTL = parsedTTL
	}

	if refreshTokenTTL := os.Getenv("REFRESH_TOKEN_TTL"); refreshTokenTTL != "" {
		parsedTTL, err := time.ParseDuration(refreshTokenTTL)
		if err != nil {
			return Config{}, fmt.Errorf("parse REFRESH_TOKEN_TTL: %w", err)
		}

		config.RefreshTokenTTL = parsedTTL
	}

	if config.DBAddress == "" {
		return Config{}, fmt.Errorf("DB_ADDRESS is required")
	}

	if config.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	return config, nil
}
