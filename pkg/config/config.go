package config

import (
	"fmt"
	"os"

	"github.com/storacha/piri/pkg/config"
	"gopkg.in/yaml.v3"
)

// Config represents the PDP server configuration
type Config struct {
	Server ServerConfig   `yaml:"server"`
	PDP    PDPConfig      `yaml:"pdp"`
	Piri   *config.Config `yaml:"piri,omitempty"` // Optional Piri integration
}

// ServerConfig represents the HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// PDPConfig represents the PDP-specific configuration
type PDPConfig struct {
	DataDir    string `yaml:"data_dir"`
	EthAddress string `yaml:"eth_address"`
	LotusURL   string `yaml:"lotus_url"`
	KeyFile    string `yaml:"key_file,omitempty"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.PDP.DataDir == "" {
		cfg.PDP.DataDir = "./data"
	}

	return &cfg, nil
}
