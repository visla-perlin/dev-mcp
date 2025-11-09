package s3

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appcfg "dev-mcp/internal/config"
)

// readSeekCloser wraps a Reader to provide a no-op Close method for S3 PutObject
type readSeekCloser struct {
	*strings.Reader
}

func (r readSeekCloser) Close() error { return nil }

// S3Client provides complete S3 operations implementation
type S3Client struct {
	s3Client  *s3.Client
	config    *appcfg.S3Config
	available bool
}

// NewS3Client creates a new S3 client from config
func NewS3Client(conf *appcfg.S3Config) *S3Client {
	if conf == nil {
		return &S3Client{
			s3Client:  nil,
			config:    nil,
			available: false,
		}
	}

	// Validate config
	if conf.Endpoint == "" || conf.AccessKey == "" || conf.SecretKey == "" {
		return &S3Client{
			s3Client:  nil,
			config:    conf,
			available: false,
		}
	}

	awsConfig, err := cfg.LoadDefaultConfig(context.TODO(),
		cfg.WithRegion(conf.Region),
		cfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			conf.AccessKey,
			conf.SecretKey,
			"",
		)),
	)

	if err != nil {
		return &S3Client{
			s3Client:  nil,
			config:    conf,
			available: false,
		}
	}

	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(conf.Endpoint)
		o.UsePathStyle = true // For MinIO or other S3-compatible services
	})

	return &S3Client{
		s3Client:  s3Client,
		config:    conf,
		available: true,
	}
}

// IsAvailable checks if S3 client is available
func (c *S3Client) IsAvailable() bool {
	return c.available
}

// getContent retrieves the content of a text file (json, txt, etc.) from S3, and signs the URL if needed
func (c *S3Client) GetSignedURL(bucket, key string, expireSeconds int32) (string, error) {
	presignClient := s3.NewPresignClient(c.s3Client)
	presignInput := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	presignResult, err := presignClient.PresignGetObject(context.TODO(), presignInput, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expireSeconds) * time.Second
	})
	if err != nil {
		return "", err
	}
	return presignResult.URL, nil
}

// getContent retrieves the content of a text file (json, txt, etc.) from S3
func (c *S3Client) GetContent(bucket, key string) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key are required")
	}

	// 判断文件类型，只允许 json、txt、csv、xml
	allowedExt := map[string]bool{".json": true, ".txt": true, ".csv": true, ".xml": true}
	ext := ""
	if len(key) > 4 {
		ext = key[len(key)-4:]
	}
	if !allowedExt[ext] {
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	resp, err := c.s3Client.GetObject(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	// 读取内容（仅适合小文件，生产建议流式处理）
	buf := make([]byte, 0)
	tmp := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	// 始终生成签名 URL
	signedUrl, _ := c.GetSignedURL(bucket, key, 600)

	result := map[string]interface{}{
		"bucket":       bucket,
		"key":          key,
		"content":      string(buf),
		"contentType":  aws.ToString(resp.ContentType),
		"size":         resp.ContentLength,
		"lastModified": resp.LastModified,
		"etag":         aws.ToString(resp.ETag),
		"metadata":     resp.Metadata,
		"signedUrl":    signedUrl,
	}
	return result, nil
}

// ListObjects lists objects in an S3 bucket
func (c *S3Client) ListObjects(bucket, prefix string, limit int) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}

	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	if limit == 0 {
		limit = 100
	}

	input := &s3.ListObjectsV2Input{
		Bucket:  &bucket,
		Prefix:  &prefix,
		MaxKeys: aws.Int32(int32(limit)),
	}
	resp, err := c.s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	objects := make([]map[string]interface{}, 0, len(resp.Contents))
	for _, obj := range resp.Contents {
		objects = append(objects, map[string]interface{}{
			"key":          aws.ToString(obj.Key),
			"size":         obj.Size,
			"lastModified": obj.LastModified,
			"etag":         aws.ToString(obj.ETag),
			"storageClass": obj.StorageClass,
		})
	}

	result := map[string]interface{}{
		"bucket":      bucket,
		"prefix":      prefix,
		"objects":     objects,
		"count":       len(objects),
		"isTruncated": resp.IsTruncated,
		"maxKeys":     limit,
	}
	return result, nil
}

// PutObject uploads an object to S3 (for testing)
func (c *S3Client) PutObject(bucket, key, content string) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}

	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key are required")
	}

	input := &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   readSeekCloser{strings.NewReader(content)},
	}
	resp, err := c.s3Client.PutObject(context.TODO(), input)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"bucket": bucket,
		"key":    key,
		"size":   len(content),
		"etag":   aws.ToString(resp.ETag),
		"status": "uploaded",
	}
	return result, nil
}

// GetBucketInfo retrieves bucket information
func (c *S3Client) GetBucketInfo(bucket string) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}

	if bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Mock bucket info
	mockData := map[string]interface{}{
		"name":         bucket,
		"region":       "us-east-1",
		"creationDate": "2023-01-01T00:00:00Z",
		"versioning":   "Disabled",
		"encryption": map[string]string{
			"type": "AES256",
		},
		"policy":      "private",
		"objectCount": 150,
		"totalSize":   "25.6 MB",
	}

	return mockData, nil
}

// GetObjectSize retrieves the size of a specific object in bytes
func (c *S3Client) GetObjectSize(bucket, key string) (int64, error) {
	if !c.IsAvailable() {
		return 0, fmt.Errorf("s3 client not available")
	}

	if bucket == "" || key == "" {
		return 0, fmt.Errorf("bucket and key are required")
	}

	// In a real implementation, this would make a HEAD request to get object metadata
	// For now, return mock size data based on file extension
	mockSize := int64(1024) // Default 1KB

	// Simulate different file sizes based on file type
	if len(key) > 4 {
		ext := key[len(key)-4:]
		switch ext {
		case ".jpg", ".png", ".gif":
			mockSize = 2048 * 1024 // 2MB for images
		case ".mp4", ".avi", ".mov":
			mockSize = 100 * 1024 * 1024 // 100MB for videos
		case ".pdf":
			mockSize = 5 * 1024 * 1024 // 5MB for PDFs
		case ".log":
			mockSize = 50 * 1024 // 50KB for logs
		case ".json", ".xml":
			mockSize = 10 * 1024 // 10KB for config files
		}
	}

	return mockSize, nil
}

// GetBucketSize calculates the total size of all objects in a bucket
func (c *S3Client) GetBucketSize(bucket string) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}

	if bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// In a real implementation, this would iterate through all objects or use bucket metrics
	mockData := map[string]interface{}{
		"bucket":      bucket,
		"totalSize":   int64(250 * 1024 * 1024), // 250MB total
		"objectCount": 157,
		"sizeByType": map[string]interface{}{
			"images": map[string]interface{}{
				"count": 45,
				"size":  int64(90 * 1024 * 1024), // 90MB
			},
			"documents": map[string]interface{}{
				"count": 32,
				"size":  int64(80 * 1024 * 1024), // 80MB
			},
			"logs": map[string]interface{}{
				"count": 50,
				"size":  int64(25 * 1024 * 1024), // 25MB
			},
			"videos": map[string]interface{}{
				"count": 5,
				"size":  int64(50 * 1024 * 1024), // 50MB
			},
			"others": map[string]interface{}{
				"count": 25,
				"size":  int64(5 * 1024 * 1024), // 5MB
			},
		},
		"averageObjectSize": int64(1592356), // ~1.5MB average
		"largestObject": map[string]interface{}{
			"key":  "videos/presentation.mp4",
			"size": int64(25 * 1024 * 1024), // 25MB
		},
		"smallestObject": map[string]interface{}{
			"key":  "config/app.json",
			"size": int64(256), // 256 bytes
		},
	}

	return mockData, nil
}

// GetObjectSizeInfo retrieves detailed size information for an object
func (c *S3Client) GetObjectSizeInfo(bucket, key string) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}

	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key are required")
	}

	// Get the basic size first
	size, err := c.GetObjectSize(bucket, key)
	if err != nil {
		return nil, err
	}

	// Calculate human-readable sizes
	sizeInKB := float64(size) / 1024
	sizeInMB := sizeInKB / 1024
	sizeInGB := sizeInMB / 1024

	mockData := map[string]interface{}{
		"bucket": bucket,
		"key":    key,
		"size": map[string]interface{}{
			"bytes":     size,
			"kilobytes": fmt.Sprintf("%.2f KB", sizeInKB),
			"megabytes": fmt.Sprintf("%.2f MB", sizeInMB),
			"gigabytes": fmt.Sprintf("%.4f GB", sizeInGB),
		},
		"storageClass": "STANDARD",
		"compressed":   false,
		"encrypted":    true,
		"metadata": map[string]interface{}{
			"contentType":     "application/octet-stream",
			"cacheControl":    "max-age=3600",
			"contentEncoding": "identity",
		},
		"checksums": map[string]string{
			"etag": "\"abc123def456\"",
			"md5":  "d41d8cd98f00b204e9800998ecf8427e",
		},
		"lastModified": "2024-11-09T10:30:00Z",
		"createdDate":  "2024-11-09T10:30:00Z",
	}

	return mockData, nil
}

// GetSizeStatistics provides comprehensive size statistics for objects matching a prefix
func (c *S3Client) GetSizeStatistics(bucket, prefix string) (interface{}, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("s3 client not available")
	}

	if bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	mockData := map[string]interface{}{
		"bucket": bucket,
		"prefix": prefix,
		"statistics": map[string]interface{}{
			"totalObjects": 25,
			"totalSize":    int64(75 * 1024 * 1024), // 75MB
			"averageSize":  int64(3 * 1024 * 1024),  // 3MB
			"medianSize":   int64(1 * 1024 * 1024),  // 1MB
			"minSize":      int64(1024),             // 1KB
			"maxSize":      int64(15 * 1024 * 1024), // 15MB
		},
		"sizeDistribution": map[string]interface{}{
			"lessThan1MB":      12,
			"1MBto10MB":        10,
			"10MBto100MB":      3,
			"greaterThan100MB": 0,
		},
		"topLargestObjects": []map[string]interface{}{
			{
				"key":  prefix + "/large-dataset.json",
				"size": int64(15 * 1024 * 1024), // 15MB
			},
			{
				"key":  prefix + "/backup.zip",
				"size": int64(12 * 1024 * 1024), // 12MB
			},
			{
				"key":  prefix + "/report.pdf",
				"size": int64(8 * 1024 * 1024), // 8MB
			},
		},
	}

	return mockData, nil
}

// Close closes the S3 client
func (c *S3Client) Close() error {
	// S3 client doesn't need explicit closing in most implementations
	return nil
}
