package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/database"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/swagger"
)

// ResourceDefinition represents a resource with its metadata and handler
type ResourceDefinition struct {
	Resource *mcp.Resource
	Handler  mcp.ResourceHandler
}

// GetAllResources collects all resources from different managers
func GetAllResources(ctx context.Context, db *database.DB, lokiClient *loki.Client, s3Client *s3.Client, swaggerClient *swagger.Client) []ResourceDefinition {
	var allResources []ResourceDefinition

	// Add database resources
	if db != nil {
		dbResources := getDatabaseResources(ctx, db)
		allResources = append(allResources, dbResources...)
		log.Printf("Added %d database resources", len(dbResources))
	}

	// Add Loki resources
	if lokiClient != nil {
		lokiResources := getLokiResources(ctx, lokiClient)
		allResources = append(allResources, lokiResources...)
		log.Printf("Added %d Loki resources", len(lokiResources))
	}

	// Add S3 resources
	if s3Client != nil {
		s3Resources := getS3Resources(ctx, s3Client)
		allResources = append(allResources, s3Resources...)
		log.Printf("Added %d S3 resources", len(s3Resources))
	}

	// Add Swagger resources
	if swaggerClient != nil {
		swaggerResources := getSwaggerResources(ctx, swaggerClient)
		allResources = append(allResources, swaggerResources...)
		log.Printf("Added %d Swagger resources", len(swaggerResources))
	}

	log.Printf("Total resources registered: %d", len(allResources))
	return allResources
}

// getDatabaseResources returns database table resources
func getDatabaseResources(ctx context.Context, db *database.DB) []ResourceDefinition {
	var resources []ResourceDefinition

	// Get list of tables
	tables, err := db.GetTables()
	if err != nil {
		log.Printf("Warning: Failed to get database tables: %v", err)
		return resources
	}

	for _, table := range tables {
		resource := &mcp.Resource{
			URI:         fmt.Sprintf("database://tables/%s", table),
			Name:        fmt.Sprintf("Table: %s", table),
			Description: fmt.Sprintf("Database table '%s' with schema and data", table),
			MIMEType:    "application/json",
		}

		handler := createTableHandler(db, table)
		resources = append(resources, ResourceDefinition{
			Resource: resource,
			Handler:  handler,
		})
	}

	return resources
}

// createTableHandler creates a handler for a specific table
func createTableHandler(db *database.DB, tableName string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		// Get table schema
		schema, err := db.GetTableSchema(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get table schema: %w", err)
		}

		// Get sample data (first 10 rows)
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 10", tableName)
		sampleData, err := db.Query(query)
		if err != nil {
			log.Printf("Warning: Failed to get sample data for table %s: %v", tableName, err)
			sampleData = []map[string]interface{}{}
		}

		result := map[string]interface{}{
			"tableName":  tableName,
			"schema":     schema,
			"sampleData": sampleData,
			"uri":        req.Params.URI,
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal table data: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}

// getLokiResources returns Loki log stream resources
func getLokiResources(ctx context.Context, client *loki.Client) []ResourceDefinition {
	var resources []ResourceDefinition

	// Get available log labels
	labels, err := client.GetLogLabels()
	if err != nil {
		log.Printf("Warning: Failed to get Loki labels: %v", err)
		// Return a default resource even if we can't get labels
		labels = []string{"default"}
	}

	for _, label := range labels {
		resource := &mcp.Resource{
			URI:         fmt.Sprintf("loki://streams/%s", label),
			Name:        fmt.Sprintf("Log Stream: %s", label),
			Description: fmt.Sprintf("Loki log stream for label '%s'", label),
			MIMEType:    "application/json",
		}

		handler := createStreamHandler(label)
		resources = append(resources, ResourceDefinition{
			Resource: resource,
			Handler:  handler,
		})
	}

	return resources
}

// createStreamHandler creates a handler for a specific log stream
func createStreamHandler(label string) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		result := map[string]interface{}{
			"label":       label,
			"uri":         req.Params.URI,
			"description": fmt.Sprintf("Log stream for label: %s", label),
			"queryHint":   fmt.Sprintf(`Use LogQL query like: {%s="value"}`, label),
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal stream data: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}

// getS3Resources returns S3 bucket resources
func getS3Resources(ctx context.Context, client *s3.Client) []ResourceDefinition {
	// Create a resource for S3 data access
	resource := &mcp.Resource{
		URI:         "s3://buckets/data",
		Name:        "S3 Data Access",
		Description: "Access JSON data from S3 buckets and objects",
		MIMEType:    "application/json",
	}

	handler := func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		result := map[string]interface{}{
			"uri":         req.Params.URI,
			"description": "S3 Data Access Resource",
			"usage":       "Use s3_query tool to retrieve specific JSON data from S3 URLs or bucket/key combinations",
			"examples": []string{
				"s3://bucket-name/path/to/file.json",
				"bucket: my-bucket, key: data/file.json",
			},
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal S3 data: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}

	return []ResourceDefinition{{
		Resource: resource,
		Handler:  handler,
	}}
}

// getSwaggerResources returns Swagger API resources
func getSwaggerResources(ctx context.Context, client *swagger.Client) []ResourceDefinition {
	spec := client.GetSpec()
	if spec == nil {
		log.Printf("Warning: No swagger specification loaded")
		return []ResourceDefinition{}
	}

	var resources []ResourceDefinition

	// Add main API spec resource
	apiResource := &mcp.Resource{
		URI:         "swagger://api/specification",
		Name:        "API Specification",
		Description: "Complete Swagger/OpenAPI specification",
		MIMEType:    "application/json",
	}

	apiHandler := func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		jsonData, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal API spec: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}

	resources = append(resources, ResourceDefinition{
		Resource: apiResource,
		Handler:  apiHandler,
	})

	return resources
}
