package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the SCITT service configuration
type Config struct {
	// Issuer is the transparency service URL
	Issuer string `yaml:"issuer"`

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
	Type      string `yaml:"type"`       // "sqlite" or "mongodb"
	Path      string `yaml:"path"`       // For SQLite database file path
	EnableWAL bool   `yaml:"enable_wal"` // For SQLite WAL mode

	// MongoDB configuration
	MongoDB *MongoDBConfig `yaml:"mongodb,omitempty"`
}

// MongoDBConfig represents MongoDB database configuration
type MongoDBConfig struct {
	URI      string `yaml:"uri"`      // MongoDB connection URI
	Database string `yaml:"database"` // Database name
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Type string `yaml:"type"` // "local", "memory", "s3", or "azure"
	Path string `yaml:"path"` // For local storage

	// S3 configuration (future use)
	S3 *S3Config `yaml:"s3,omitempty"`

	// Azure Blob Storage configuration
	Azure *AzureConfig `yaml:"azure,omitempty"`
}

// S3Config represents S3/MinIO storage configuration
type S3Config struct {
	Endpoint  string `yaml:"endpoint"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	UseSSL    bool   `yaml:"use_ssl"`
}

// AzureConfig represents Azure Blob Storage configuration
type AzureConfig struct {
	// Storage account name (optional if using SAS URL)
	AccountName string `yaml:"account_name,omitempty"`

	// Container name where tiles will be stored
	Container string `yaml:"container"`

	// Authentication method: "sas" or "key"
	// SAS URL: Shared Access Signature URL (recommended for security)
	// Key: Storage account key
	SASURL string `yaml:"sas_url,omitempty"`

	// AccountKey for authentication (less secure than SAS)
	AccountKey string `yaml:"account_key,omitempty"`
}

// KeysConfig represents service key configuration
type KeysConfig struct {
	Private string `yaml:"private"` // Path to private key (PEM)
	Public  string `yaml:"public"`  // Path to public key (JWK)
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host   string     `yaml:"host"`
	Port   int        `yaml:"port"`
	APIKey string     `yaml:"api_key"`
	CORS   CORSConfig `yaml:"cors"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// expandEnvVars expands environment variable references in the format ${VAR_NAME}
// Returns the expanded string with environment variables resolved
func expandEnvVars(s string) string {
	// Match ${VAR_NAME} pattern
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		// Get environment variable value, return original if not found
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Return original if env var not set
	})
}

// expandConfigEnvVars recursively expands environment variables in config
func (c *Config) expandConfigEnvVars() {
	// Expand database configuration
	if c.Database.Type == "mongodb" && c.Database.MongoDB != nil {
		c.Database.MongoDB.URI = expandEnvVars(c.Database.MongoDB.URI)
		c.Database.MongoDB.Database = expandEnvVars(c.Database.MongoDB.Database)
	}

	// Expand storage configuration
	if c.Storage.Type == "azure" && c.Storage.Azure != nil {
		c.Storage.Azure.AccountName = expandEnvVars(c.Storage.Azure.AccountName)
		c.Storage.Azure.Container = expandEnvVars(c.Storage.Azure.Container)
		c.Storage.Azure.SASURL = expandEnvVars(c.Storage.Azure.SASURL)
		c.Storage.Azure.AccountKey = expandEnvVars(c.Storage.Azure.AccountKey)
	}

	// Expand server API key
	c.Server.APIKey = expandEnvVars(c.Server.APIKey)
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

	// Expand environment variable references
	config.expandConfigEnvVars()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}

	// Validate database configuration
	if c.Database.Type == "" {
		c.Database.Type = "sqlite" // Default to sqlite
	}

	if c.Database.Type != "sqlite" && c.Database.Type != "mongodb" {
		return fmt.Errorf("database type must be 'sqlite' or 'mongodb', got: %s", c.Database.Type)
	}

	if c.Database.Type == "sqlite" && c.Database.Path == "" {
		return fmt.Errorf("database path is required for SQLite")
	}

	if c.Database.Type == "mongodb" && c.Database.MongoDB == nil {
		return fmt.Errorf("mongodb configuration is required for MongoDB database type")
	}

	if c.Database.Type == "mongodb" {
		if c.Database.MongoDB.URI == "" {
			return fmt.Errorf("mongodb.uri is required for MongoDB database type")
		}
		if c.Database.MongoDB.Database == "" {
			return fmt.Errorf("mongodb.database is required for MongoDB database type")
		}
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

	if c.Storage.Type == "azure" {
		if c.Storage.Azure == nil {
			return fmt.Errorf("Azure configuration is required for Azure storage")
		}
		if c.Storage.Azure.Container == "" {
			return fmt.Errorf("azure.container is required for Azure storage")
		}
		// Require either SAS URL or account name + key
		if c.Storage.Azure.SASURL == "" && (c.Storage.Azure.AccountName == "" || c.Storage.Azure.AccountKey == "") {
			return fmt.Errorf("azure.sas_url OR (azure.account_name AND azure.account_key) is required")
		}
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
		Issuer: "http://127.0.0.1:56177",
		Database: DatabaseConfig{
			Type:      "sqlite",
			Path:      "./demo/scitt.db",
			EnableWAL: true,
		},
		Storage: StorageConfig{
			Type: "local",
			Path: "./demo/tiles",
		},
		Keys: KeysConfig{
			Private: "./demo/priv.cbor",
			Public:  "./demo/pub.cbor",
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 56177,
			CORS: CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}
}

// GenerateAPIKey generates a cryptographically secure random API key
// Returns a 64-character hexadecimal string (32 bytes of randomness)
func GenerateAPIKey() (string, error) {
	// Generate 32 bytes of cryptographically secure random data
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Encode as 64-character hexadecimal string
	return hex.EncodeToString(randomBytes), nil
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
