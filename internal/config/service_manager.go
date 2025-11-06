package config

import (
	"fmt"
	"time"
)

// ServiceManager manages the status of all configured services
type ServiceManager struct {
	config     *Config
	validation *ValidationResult
	lastCheck  time.Time
}

// NewServiceManager creates a new service manager
func NewServiceManager(config *Config) *ServiceManager {
	return &ServiceManager{
		config: config,
	}
}

// CheckAllServices performs a comprehensive check of all services
func (sm *ServiceManager) CheckAllServices() *ValidationResult {
	sm.validation = sm.config.ValidateConfig()
	sm.lastCheck = time.Now()
	return sm.validation
}

// GetServiceStatus returns the status of a specific service
func (sm *ServiceManager) GetServiceStatus(serviceName string) (*ConfigStatus, error) {
	if sm.validation == nil {
		sm.CheckAllServices()
	}

	for _, service := range sm.validation.Services {
		if service.Service == serviceName {
			return &service, nil
		}
	}

	return nil, fmt.Errorf("service '%s' not found", serviceName)
}

// RequireService checks if a service is properly configured and returns an error if not
func (sm *ServiceManager) RequireService(serviceName string) error {
	status, err := sm.GetServiceStatus(serviceName)
	if err != nil {
		return err
	}

	if !status.Configured {
		return fmt.Errorf("service '%s' is not properly configured: %s", serviceName, status.Message)
	}

	return nil
}

// GetConfiguredServices returns a list of all properly configured services
func (sm *ServiceManager) GetConfiguredServices() []string {
	if sm.validation == nil {
		sm.CheckAllServices()
	}

	var configured []string
	for _, service := range sm.validation.Services {
		if service.Configured {
			configured = append(configured, service.Service)
		}
	}

	return configured
}

// GetUnconfiguredServices returns a list of all unconfigured services
func (sm *ServiceManager) GetUnconfiguredServices() []string {
	if sm.validation == nil {
		sm.CheckAllServices()
	}

	var unconfigured []string
	for _, service := range sm.validation.Services {
		if !service.Configured {
			unconfigured = append(unconfigured, service.Service)
		}
	}

	return unconfigured
}

// PrintServiceStatus prints a formatted status report of all services
func (sm *ServiceManager) PrintServiceStatus() {
	if sm.validation == nil {
		sm.CheckAllServices()
	}

	fmt.Println("=== Service Configuration Status ===")
	fmt.Printf("Last checked: %s\n", sm.lastCheck.Format(time.RFC3339))
	fmt.Printf("Overall status: %v\n\n", sm.validation.Valid)

	for _, service := range sm.validation.Services {
		status := "❌"
		if service.Configured {
			status = "✅"
		}

		required := ""
		if service.Required {
			required = " (Required)"
		}

		fmt.Printf("%s %s%s: %s\n", status, service.Service, required, service.Message)
	}

	if len(sm.validation.Errors) > 0 {
		fmt.Println("\n🚨 Configuration Errors:")
		for _, err := range sm.validation.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(sm.validation.Warnings) > 0 {
		fmt.Println("\n⚠️  Configuration Warnings:")
		for _, warning := range sm.validation.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	fmt.Println()
}
