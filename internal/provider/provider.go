package provider

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProviderClient defines the interface for MCP provider clients
type ProviderClient interface {
	// Test tests the configuration and connection
	Test(config interface{}) error

	// AddTools adds tools to the MCP server if test passes
	AddTools(server *mcp.Server, config interface{}) error
}

// ResourceDefinition represents a resource with its metadata and handler
// Simplified to generic interface for now - can be expanded later
type ResourceDefinition struct {
	Resource interface{}
	Handler  interface{}
}

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name      string
	available bool
	status    ProviderStatus
}

type ProviderStatus struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
	Error     string `json:"error,omitempty"`
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string) *BaseProvider {
	return &BaseProvider{
		name:      name,
		available: false,
	}
}

// Name returns the provider name
func (bp *BaseProvider) Name() string {
	return bp.name
}

// IsAvailable returns whether the provider is available
func (bp *BaseProvider) IsAvailable() bool {
	return bp.available
}

// SetAvailable sets the availability status
func (bp *BaseProvider) SetAvailable(available bool) {
	bp.available = available
}

func (bp *BaseProvider) SetStatus(available bool, message string, err error) {
	bp.available = available
	bp.status.Available = available
	bp.status.Message = message
	if err != nil {
		bp.status.Error = err.Error()
	} else {
		bp.status.Error = ""
	}
}

// Close provides default close implementation (can be overridden)
func (bp *BaseProvider) Close() error {
	return nil
}
