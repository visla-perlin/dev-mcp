package s3

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"dev-mcp/internal/config"
)

// Client represents an S3 client
type Client struct {
	s3Client *s3.S3
	config   *config.S3Config
}

// New creates a new S3 client
func New(cfg *config.S3Config) (*Client, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Endpoint: aws.String(cfg.Endpoint),
		Region:   aws.String(cfg.Region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 client
	s3Client := s3.New(sess)

	return &Client{
		s3Client: s3Client,
		config:   cfg,
	}, nil
}

// GetJSONFromURL retrieves and parses JSON data from an S3 URL
func (c *Client) GetJSONFromURL(url string) (map[string]interface{}, error) {
	// Parse S3 URL
	bucket, key, err := parseS3URL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 URL: %w", err)
	}

	// Get object from S3
	result, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Parse JSON
	var jsonData map[string]interface{}
	if err := json.NewDecoder(result.Body).Decode(&jsonData); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return jsonData, nil
}

// GetJSONFromBucketAndKey retrieves and parses JSON data from S3 bucket and key
func (c *Client) GetJSONFromBucketAndKey(bucket, key string) (map[string]interface{}, error) {
	// Get object from S3
	result, err := c.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Parse JSON
	var jsonData map[string]interface{}
	if err := json.NewDecoder(result.Body).Decode(&jsonData); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return jsonData, nil
}

// parseS3URL parses an S3 URL into bucket and key
func parseS3URL(url string) (bucket, key string, err error) {
	// Remove s3:// prefix
	url = strings.TrimPrefix(url, "s3://")

	// Split bucket and key
	parts := strings.SplitN(url, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid S3 URL format")
	}

	return parts[0], parts[1], nil
}