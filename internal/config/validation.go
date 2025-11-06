package config

import (
	"fmt"
	"strings"
)

// ConfigStatus represents the configuration status of a service
type ConfigStatus struct {
	Service    string `json:"service"`
	Configured bool   `json:"configured"`
	Required   bool   `json:"required"`
	Message    string `json:"message"`
}

// ValidationResult represents the overall configuration validation result
type ValidationResult struct {
	Valid    bool           `json:"valid"`
	Services []ConfigStatus `json:"services"`
	Errors   []string       `json:"errors"`
	Warnings []string       `json:"warnings"`
}

// ValidateConfig validates the configuration and returns detailed status
func (c *Config) ValidateConfig() *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Services: []ConfigStatus{},
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate Database Configuration
	dbStatus := c.validateDatabaseConfig()
	result.Services = append(result.Services, dbStatus)
	if !dbStatus.Configured && dbStatus.Required {
		result.Valid = false
		result.Errors = append(result.Errors, dbStatus.Message)
	}

	// Validate Loki Configuration
	lokiStatus := c.validateLokiConfig()
	result.Services = append(result.Services, lokiStatus)
	if !lokiStatus.Configured {
		result.Warnings = append(result.Warnings, lokiStatus.Message)
	}

	// Validate S3 Configuration
	s3Status := c.validateS3Config()
	result.Services = append(result.Services, s3Status)
	if !s3Status.Configured {
		result.Warnings = append(result.Warnings, s3Status.Message)
	}

	// Validate Sentry Configuration
	sentryStatus := c.validateSentryConfig()
	result.Services = append(result.Services, sentryStatus)
	if !sentryStatus.Configured {
		result.Warnings = append(result.Warnings, sentryStatus.Message)
	}

	// Validate Swagger Configuration
	swaggerStatus := c.validateSwaggerConfig()
	result.Services = append(result.Services, swaggerStatus)
	if !swaggerStatus.Configured {
		result.Warnings = append(result.Warnings, swaggerStatus.Message)
	}

	// Validate Auth Configuration
	authStatus := c.validateAuthConfig()
	result.Services = append(result.Services, authStatus)
	if !authStatus.Configured {
		result.Warnings = append(result.Warnings, authStatus.Message)
	}

	return result
}

// validateDatabaseConfig validates database configuration
func (c *Config) validateDatabaseConfig() ConfigStatus {
	status := ConfigStatus{
		Service:  "database",
		Required: true, // Database is required for core functionality
	}

	missing := []string{}
	if c.Database.Host == "" {
		missing = append(missing, "host")
	}
	if c.Database.Port == 0 {
		missing = append(missing, "port")
	}
	if c.Database.Username == "" {
		missing = append(missing, "username")
	}
	if c.Database.Password == "" {
		missing = append(missing, "password")
	}
	if c.Database.DBName == "" {
		missing = append(missing, "dbname")
	}

	if len(missing) > 0 {
		status.Configured = false
		status.Message = fmt.Sprintf("Database not configured: missing %s", strings.Join(missing, ", "))
	} else {
		status.Configured = true
		status.Message = "Database configuration is complete"
	}

	return status
}

// validateLokiConfig validates Loki configuration
func (c *Config) validateLokiConfig() ConfigStatus {
	status := ConfigStatus{
		Service:  "loki",
		Required: false,
	}

	if c.Loki.Host == "" {
		status.Configured = false
		status.Message = "Loki not configured: missing host"
	} else {
		status.Configured = true
		status.Message = "Loki configuration is complete"
	}

	return status
}

// validateS3Config validates S3 configuration
func (c *Config) validateS3Config() ConfigStatus {
	status := ConfigStatus{
		Service:  "s3",
		Required: false,
	}

	missing := []string{}
	if c.S3.Region == "" {
		missing = append(missing, "region")
	}
	if c.S3.AccessKey == "" {
		missing = append(missing, "access_key")
	}
	if c.S3.SecretKey == "" {
		missing = append(missing, "secret_key")
	}

	if len(missing) > 0 {
		status.Configured = false
		status.Message = fmt.Sprintf("S3 not configured: missing %s", strings.Join(missing, ", "))
	} else {
		status.Configured = true
		status.Message = "S3 configuration is complete"
	}

	return status
}

// validateSentryConfig validates Sentry configuration
func (c *Config) validateSentryConfig() ConfigStatus {
	status := ConfigStatus{
		Service:  "sentry",
		Required: false,
	}

	if c.Sentry.DSN == "" {
		status.Configured = false
		status.Message = "Sentry not configured: missing DSN"
	} else {
		status.Configured = true
		status.Message = "Sentry configuration is complete"
	}

	return status
}

// validateSwaggerConfig validates Swagger configuration
func (c *Config) validateSwaggerConfig() ConfigStatus {
	status := ConfigStatus{
		Service:  "swagger",
		Required: false,
	}

	if c.Swagger.URL == "" && c.Swagger.Filepath == "" {
		status.Configured = false
		status.Message = "Swagger not configured: missing URL or file path"
	} else {
		status.Configured = true
		status.Message = "Swagger configuration is complete"
	}

	return status
}

// validateAuthConfig validates authentication configuration
func (c *Config) validateAuthConfig() ConfigStatus {
	status := ConfigStatus{
		Service:  "auth",
		Required: false,
	}

	if !c.Auth.Enabled {
		status.Configured = false
		status.Message = "Authentication is disabled"
	} else if len(c.Auth.APIKeys) == 0 {
		status.Configured = false
		status.Message = "Authentication enabled but no API keys configured"
	} else {
		enabledKeys := 0
		for _, key := range c.Auth.APIKeys {
			if key.Enabled {
				enabledKeys++
			}
		}
		if enabledKeys == 0 {
			status.Configured = false
			status.Message = "Authentication enabled but no API keys are enabled"
		} else {
			status.Configured = true
			status.Message = fmt.Sprintf("Authentication configured with %d active API keys", enabledKeys)
		}
	}

	return status
}

// IsServiceConfigured checks if a specific service is properly configured
func (c *Config) IsServiceConfigured(serviceName string) bool {
	validation := c.ValidateConfig()
	for _, service := range validation.Services {
		if service.Service == serviceName {
			return service.Configured
		}
	}
	return false
}

// GetConfigurationErrors returns a list of configuration errors
func (c *Config) GetConfigurationErrors() []string {
	validation := c.ValidateConfig()
	return validation.Errors
}

// GetConfigurationWarnings returns a list of configuration warnings
func (c *Config) GetConfigurationWarnings() []string {
	validation := c.ValidateConfig()
	return validation.Warnings
}
