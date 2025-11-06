package swagger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"dev-mcp/internal/config"
)

// Spec represents a Swagger/OpenAPI specification
type Spec struct {
	Swagger     string                 `json:"swagger"`
	Info        Info                   `json:"info"`
	Paths       map[string]PathItem    `json:"paths"`
	Definitions map[string]Schema      `json:"definitions"`
	Schemes     []string               `json:"schemes"`
	Host        string                 `json:"host"`
	BasePath    string                 `json:"basePath"`
	Data        map[string]interface{} `json:"-"` // Raw data
}

// Info represents the info section of a Swagger spec
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// PathItem represents a path item in a Swagger spec
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
}

// Operation represents an operation in a Swagger spec
type Operation struct {
	Summary     string              `json:"summary"`
	Description string              `json:"description"`
	OperationID string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
	Tags        []string            `json:"tags"`
}

// Parameter represents a parameter in a Swagger spec
type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Response represents a response in a Swagger spec
type Response struct {
	Description string `json:"description"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Schema represents a schema in a Swagger spec
type Schema struct {
	Type       string                 `json:"type"`
	Properties map[string]Property    `json:"properties"`
	Required   []string               `json:"required"`
	Ref        string                 `json:"$ref"`
	Items      *Schema                `json:"items"`
	Data       map[string]interface{} `json:"-"` // Raw data
}

// Property represents a property in a schema
type Property struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Format      string  `json:"format"`
	Ref         string  `json:"$ref"`
	Items       *Schema `json:"items"`
}

// Client represents a Swagger client
type Client struct {
	config *config.SwaggerConfig
	spec   *Spec
}

// New creates a new Swagger client
func New(cfg *config.SwaggerConfig) (*Client, error) {
	client := &Client{
		config: cfg,
	}

	// Load the spec if filepath is provided
	if cfg.Filepath != "" {
		err := client.LoadSpecFromFile(cfg.Filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to load spec from file: %w", err)
		}
	}

	return client, nil
}

// LoadSpecFromFile loads a Swagger spec from a file
func (c *Client) LoadSpecFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read Swagger file: %w", err)
	}

	var spec Spec
	err = json.Unmarshal(data, &spec)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Swagger spec: %w", err)
	}

	// Store raw data
	var rawData map[string]interface{}
	err = json.Unmarshal(data, &rawData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal raw Swagger data: %w", err)
	}
	spec.Data = rawData

	c.spec = &spec
	return nil
}

// LoadSpecFromURL loads a Swagger spec from a URL
func (c *Client) LoadSpecFromURL(url string) error {
	// In a real implementation, this would make an HTTP request to fetch the spec
	// For now, we'll return an error as a placeholder
	return fmt.Errorf("not implemented: LoadSpecFromURL")
}

// GetSpec returns the loaded Swagger spec
func (c *Client) GetSpec() *Spec {
	return c.spec
}

// GetPaths returns all paths in the spec
func (c *Client) GetPaths() map[string]PathItem {
	if c.spec == nil {
		return nil
	}
	return c.spec.Paths
}

// GetPath returns a specific path item
func (c *Client) GetPath(path string) *PathItem {
	if c.spec == nil || c.spec.Paths == nil {
		return nil
	}
	if pathItem, exists := c.spec.Paths[path]; exists {
		return &pathItem
	}
	return nil
}

// GetOperations returns all operations in the spec
func (c *Client) GetOperations() map[string]map[string]*Operation {
	operations := make(map[string]map[string]*Operation)

	if c.spec == nil || c.spec.Paths == nil {
		return operations
	}

	for path, pathItem := range c.spec.Paths {
		operations[path] = make(map[string]*Operation)

		if pathItem.Get != nil {
			operations[path]["get"] = pathItem.Get
		}
		if pathItem.Post != nil {
			operations[path]["post"] = pathItem.Post
		}
		if pathItem.Put != nil {
			operations[path]["put"] = pathItem.Put
		}
		if pathItem.Delete != nil {
			operations[path]["delete"] = pathItem.Delete
		}
		if pathItem.Options != nil {
			operations[path]["options"] = pathItem.Options
		}
		if pathItem.Head != nil {
			operations[path]["head"] = pathItem.Head
		}
		if pathItem.Patch != nil {
			operations[path]["patch"] = pathItem.Patch
		}
	}

	return operations
}

// FindOperationsByTag finds operations with a specific tag
func (c *Client) FindOperationsByTag(tag string) []*Operation {
	var operations []*Operation

	if c.spec == nil || c.spec.Paths == nil {
		return operations
	}

	for _, pathItem := range c.spec.Paths {
		ops := []*Operation{pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Delete, pathItem.Options, pathItem.Head, pathItem.Patch}
		for _, op := range ops {
			if op != nil && containsTag(op.Tags, tag) {
				operations = append(operations, op)
			}
		}
	}

	return operations
}

// containsTag checks if a slice of tags contains a specific tag
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

// GetDefinitions returns all definitions in the spec
func (c *Client) GetDefinitions() map[string]Schema {
	if c.spec == nil {
		return nil
	}
	return c.spec.Definitions
}