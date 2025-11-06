package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Loki    LokiConfig    `yaml:"loki"`
	S3      S3Config      `yaml:"s3"`
	Sentry  SentryConfig  `yaml:"sentry"`
	Swagger SwaggerConfig `yaml:"swagger"`
	LLM     LLMConfig     `yaml:"llm"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// LokiConfig represents the Grafana Loki configuration
type LokiConfig struct {
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// S3Config represents the S3 configuration
type S3Config struct {
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
}

// SentryConfig represents the Sentry configuration
type SentryConfig struct {
	DSN         string `yaml:"dsn"`
	Environment string `yaml:"environment"`
	Release     string `yaml:"release"`
}

// SwaggerConfig represents the Swagger configuration
type SwaggerConfig struct {
	URL      string `yaml:"url"`
	Filepath string `yaml:"filepath"`
}

// LLMConfig represents the configuration for large language models
type LLMConfig struct {
	Providers []ProviderConfig `yaml:"providers"`
}

// ProviderConfig represents the configuration for a specific provider
type ProviderConfig struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	APIKey   string `yaml:"api_key"`
	Endpoint string `yaml:"endpoint"`
	Enabled  bool   `yaml:"enabled"`
	Model    string `yaml:"model"`
}

// Load loads the configuration from a file and overrides with environment variables
func Load(filepath string) (*Config, error) {
	// Load from file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with environment variables
	config.overrideWithEnv()

	return &config, nil
}

// overrideWithEnv overrides configuration with environment variables
func (c *Config) overrideWithEnv() {
	// Server configuration
	if port := os.Getenv("MCP_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}
	if host := os.Getenv("MCP_SERVER_HOST"); host != "" {
		c.Server.Host = host
	}

	// Database configuration
	if host := os.Getenv("MCP_DATABASE_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("MCP_DATABASE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Database.Port = p
		}
	}
	if username := os.Getenv("MCP_DATABASE_USERNAME"); username != "" {
		c.Database.Username = username
	}
	if password := os.Getenv("MCP_DATABASE_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbname := os.Getenv("MCP_DATABASE_DBNAME"); dbname != "" {
		c.Database.DBName = dbname
	}

	// Loki configuration
	if host := os.Getenv("MCP_LOKI_HOST"); host != "" {
		c.Loki.Host = host
	}
	if username := os.Getenv("MCP_LOKI_USERNAME"); username != "" {
		c.Loki.Username = username
	}
	if password := os.Getenv("MCP_LOKI_PASSWORD"); password != "" {
		c.Loki.Password = password
	}

	// S3 configuration
	if endpoint := os.Getenv("MCP_S3_ENDPOINT"); endpoint != "" {
		c.S3.Endpoint = endpoint
	}
	if region := os.Getenv("MCP_S3_REGION"); region != "" {
		c.S3.Region = region
	}
	if accessKey := os.Getenv("MCP_S3_ACCESS_KEY"); accessKey != "" {
		c.S3.AccessKey = accessKey
	}
	if secretKey := os.Getenv("MCP_S3_SECRET_KEY"); secretKey != "" {
		c.S3.SecretKey = secretKey
	}
	if bucket := os.Getenv("MCP_S3_BUCKET"); bucket != "" {
		c.S3.Bucket = bucket
	}

	// Sentry configuration
	if dsn := os.Getenv("MCP_SENTRY_DSN"); dsn != "" {
		c.Sentry.DSN = dsn
	}
	if env := os.Getenv("MCP_SENTRY_ENVIRONMENT"); env != "" {
		c.Sentry.Environment = env
	}
	if release := os.Getenv("MCP_SENTRY_RELEASE"); release != "" {
		c.Sentry.Release = release
	}

	// Swagger configuration
	if url := os.Getenv("MCP_SWAGGER_URL"); url != "" {
		c.Swagger.URL = url
	}
	if filepath := os.Getenv("MCP_SWAGGER_FILEPATH"); filepath != "" {
		c.Swagger.Filepath = filepath
	}

	// LLM configuration
	c.overrideLLMConfigWithEnv()
}

// overrideLLMConfigWithEnv overrides LLM configuration with environment variables
func (c *Config) overrideLLMConfigWithEnv() {
	for i := range c.LLM.Providers {
		prefix := fmt.Sprintf("MCP_LLM_PROVIDERS_%d_", i)

		if name := os.Getenv(prefix + "NAME"); name != "" {
			c.LLM.Providers[i].Name = name
		}
		if providerType := os.Getenv(prefix + "TYPE"); providerType != "" {
			c.LLM.Providers[i].Type = providerType
		}
		if apiKey := os.Getenv(prefix + "API_KEY"); apiKey != "" {
			c.LLM.Providers[i].APIKey = apiKey
		}
		if endpoint := os.Getenv(prefix + "ENDPOINT"); endpoint != "" {
			c.LLM.Providers[i].Endpoint = endpoint
		}
		if enabled := os.Getenv(prefix + "ENABLED"); enabled != "" {
			if strings.ToLower(enabled) == "true" || enabled == "1" {
				c.LLM.Providers[i].Enabled = true
			} else {
				c.LLM.Providers[i].Enabled = false
			}
		}
		if model := os.Getenv(prefix + "MODEL"); model != "" {
			c.LLM.Providers[i].Model = model
		}
	}
}