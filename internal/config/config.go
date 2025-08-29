package config

import "os"

type Config struct {
	PGHost     string
	PGPort     string
	PGUser     string
	PGPassword string
	PGDBName   string
	PGSSLmode  string
}

func LoadConfig() (*Config, error) {
	return &Config{

		PGHost:     os.Getenv("POSTGRES_HOST"),
		PGPort:     os.Getenv("POSTGRES_PORT"),
		PGUser:     os.Getenv("POSTGRES_USER"),
		PGPassword: os.Getenv("POSTGRES_PASSWORD"),
		PGDBName:   os.Getenv("POSTGRES_DBNAME"),
		PGSSLmode:  "disable", // Default
	}, nil
}
