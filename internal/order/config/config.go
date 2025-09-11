package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// if only yaml is provided, use that config path
// const configPath = "config.yaml"

type Config struct {
	DB  *Postgres `yaml:"database"`
	RMQ *RabbitMQ `yaml:"rabbitmq"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RabbitMQ struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	VHost    string `yaml:"vhost"`
}

func LoadConfig(configPath string) (*Config, error) {
	// Only if yaml is allowed, we would use that code below
	//
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadDotEnv() *Config {
	return &Config{
		DB: &Postgres{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "admin"),
			Password: getEnv("POSTGRES_PASSWORD", "admin"),
			Database: getEnv("POSTGRES_DBNAME", "restaurant_db"),
		},
		RMQ: &RabbitMQ{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnv("RABBITMQ_PORT_APP", "5672"),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:    getEnv("RABBITMQ_VHOST", "five-nights-at-freddys"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
