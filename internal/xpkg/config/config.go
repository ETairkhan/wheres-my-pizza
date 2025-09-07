package config

import (
	"os"
	"strings"
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
	cnf := &Config{}
	err = simpleYAMLUnmarshal(data, cnf)
	if err != nil {
		return nil, err
	}
	return cnf, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// simpleYAMLUnmarshal parses a basic YAML structure into the config object
func simpleYAMLUnmarshal(data []byte, config *Config) error {
	lines := strings.Split(string(data), "\n")
	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Check for section headers
		if strings.HasSuffix(line, ":") {
			currentSection = strings.TrimSuffix(line, ":")
			continue
		}

		// Parse key-value pairs
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove quotes if present
			value = strings.Trim(value, `"'`)

			switch currentSection {
			case "database":
				setPostgresField(config.DB, key, value)
			case "rabbitmq":
				setRabbitMQField(config.RMQ, key, value)
			}
		}
	}

	return nil
}

func setPostgresField(pg *Postgres, key, value string) {
	switch key {
	case "host":
		pg.Host = value
	case "port":
		pg.Port = value
	case "user":
		pg.User = value
	case "password":
		pg.Password = value
	case "database":
		pg.Database = value
	}
}

func setRabbitMQField(rmq *RabbitMQ, key, value string) {
	switch key {
	case "host":
		rmq.Host = value
	case "port":
		rmq.Port = value
	case "user":
		rmq.User = value
	case "password":
		rmq.Password = value
	case "vhost":
		rmq.VHost = value
	}
}