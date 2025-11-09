package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/provider/loki"
	"dev-mcp/internal/provider/s3"
)

// ResourceDefinition represents a resource with its metadata and handler
type ResourceDefinition struct {
	Resource *mcp.Resource
	Handler  mcp.ResourceHandler
}

// GetAllResources collects all resources from different managers
func GetAllResources(ctx context.Context, db interface{}, lokiClient *loki.Client, s3Client *s3.S3Client) []ResourceDefinition {
	var allResources []ResourceDefinition

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

	log.Printf("Total resources registered: %d", len(allResources))
	return allResources
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
func getS3Resources(ctx context.Context, client *s3.S3Client) []ResourceDefinition {
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
