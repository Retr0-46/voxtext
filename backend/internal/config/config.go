package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseAddress string
	ServerAddress   string
	JWTSecret       string
}

func Load() (Config, error) {
	if err := godotenv.Load(".env"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	databaseAddress, err := required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	jwtSecret, err := required("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	serverAddress := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if serverAddress == "" {
		serverAddress = ":8080"
	}

	return Config{
		DatabaseAddress: databaseAddress,
		ServerAddress:   serverAddress,
		JWTSecret:       jwtSecret,
	}, nil
}

func required(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}
