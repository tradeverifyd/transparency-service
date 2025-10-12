package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the SCITT service configuration
type Config struct {
	// Origin is the transparency service URL
	Origin string `yaml:"origin"`

	// Database configuration
	Database DatabaseConfig `yaml:"database"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage"`

	// Service keys
	Keys KeysConfig `yaml:"keys"`

	// HTTP server configuration
	Server ServerConfig `yaml:"server"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path      string `yaml:"path"`
	EnableWAL bool   `yaml:"enable_wal"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type string `yaml:"type"` // "local", "memory", or "s3"
	Path string `yaml:"path"` // For local storage

	// S3 configuration (future use)
	S3 *S3Config `yaml:"s3,omitempty"`
}

// S3Config represents S3/MinIO storage configuration
type S3Config struct {
	Endpoint  string `yaml:"endpoint"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	UseSSL    bool   `yaml:"use_ssl"`
}

// KeysConfig represents service key configuration
type KeysConfig struct {
	Private string `yaml:"private"` // Path to private key (PEM)
	Public  string `yaml:"public"`  // Path to public key (JWK)
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host string     `yaml:"host"`
	Port int        `yaml:"port"`
	CORS CORSConfig `yaml:"cors"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Origin == "" {
		return fmt.Errorf("origin is required")
	}

	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}

	if c.Storage.Type == "" {
		return fmt.Errorf("storage type is required")
	}

	if c.Storage.Type == "local" && c.Storage.Path == "" {
		return fmt.Errorf("storage path is required for local storage")
	}

	if c.Storage.Type == "s3" && c.Storage.S3 == nil {
		return fmt.Errorf("S3 configuration is required for S3 storage")
	}

	if c.Keys.Private == "" {
		return fmt.Errorf("private key path is required")
	}

	if c.Keys.Public == "" {
		return fmt.Errorf("public key path is required")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Origin: "https://transparency.example.com",
		Database: DatabaseConfig{
			Path:      "scitt.db",
			EnableWAL: true,
		},
		Storage: StorageConfig{
			Type: "local",
			Path: "./storage",
		},
		Keys: KeysConfig{
			Private: "service-key.pem",
			Public:  "service-key.jwk",
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			CORS: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}
}

// SaveConfig saves configuration to a YAML file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
