package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBURL           string
	MIGRAURL        string
	ListenAddr      string
	JWTSecret       string
	MWSTablesURL    string
	MWSTablesAPIKey string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DBURL:        strings.TrimSpace(os.Getenv("DB_URL")),
		MIGRAURL:     strings.TrimSpace(os.Getenv("MIGRA_URL")),
		ListenAddr:   strings.TrimSpace(os.Getenv("LISTEN_ADDR")),
		JWTSecret:    strings.TrimSpace(os.Getenv("JWT_SECRET")),
		MWSTablesURL:    strings.TrimSpace(os.Getenv("MWS_TABLES_URL")),
		MWSTablesAPIKey: strings.TrimSpace(os.Getenv("MWS_TABLES_API_KEY")),
	}

	return cfg, nil
}
