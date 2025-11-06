package main

import (
	"fmt"
	"log"
	"os"

	"dev-mcp/internal/config"
	"dev-mcp/internal/database"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/mcp/server"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

func main() {
	// Check for debug mode (for future use)
	for _, arg := range os.Args {
		if arg == "--debug" || arg == "-d" {
			fmt.Println("Debug mode enabled")
			break
		}
	}

	// Parse command line arguments for transport mode
	transportMode := "sse" // Force SSE mode as default
	var mcpMode bool

	for i, arg := range os.Args {
		if arg == "mcp" {
			mcpMode = true
		} else if arg == "--transport" || arg == "-t" {
			if i+1 < len(os.Args) {
				transportMode = os.Args[i+1]
			}
		} else if arg == "--sse" {
			transportMode = "sse"
		} else if arg == "--http" {
			transportMode = "http"
		} else if arg == "--stdio" {
			// Force SSE even for stdio requests
			transportMode = "sse"
			fmt.Println("Note: stdio mode forced to SSE mode")
		}
	}

	// Check if we should start in MCP mode
	if mcpMode {
		startMCPServer(transportMode)
		return
	} // Load configuration for testing/demo mode
	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("Dev MCP initialized successfully!")
	fmt.Printf("Configuration loaded from config.yaml\n")

	// Run health check to verify all services
	if err := healthCheck(cfg); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("✓ All services health check passed")
	}

	fmt.Println("\nDev MCP is ready to use!")
	fmt.Println("Available services:")
	fmt.Println("  - Database queries (MySQL)")
	fmt.Println("  - Grafana Loki log queries")
	fmt.Println("  - S3 JSON data access")
	fmt.Println("  - Sentry error tracking")
	fmt.Println("  - Swagger API specification parsing")
	fmt.Println("  - Large Language Models (LLM) service")
	fmt.Println("  - HTTP request simulation")
	fmt.Println("\nTo start as MCP server, run: go run cmd/main.go mcp")

	// Exit gracefully instead of infinite wait
	return
}

// startMCPServer initializes and starts the MCP server with transport mode support
func startMCPServer(transportMode string) {
	fmt.Printf("🚀 Starting MCP server with transport mode: %s\n", transportMode)

	// Force SSE mode as requested
	if transportMode != "sse" {
		fmt.Printf("Note: %s mode forced to SSE mode\n", transportMode)
		transportMode = "sse"
	}

	// Load configuration using existing API
	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Starting Dev MCP Server...")

	// Initialize all services
	services := server.InitializeServices(cfg)
	defer services.Close()

	// Create the enhanced MCP server using our refactored implementation
	mcpServer := server.NewMCPServer(cfg, services)

	log.Printf("🚀 Starting MCP server with %s transport mode...\n", transportMode)
	log.Printf("Authentication enabled: %t\n", cfg.Auth.Enabled)

	// Start server with the specified transport mode
	if err := mcpServer.Start(transportMode); err != nil {
		log.Fatalf("MCP server failed: %v", err)
	}
} // healthCheck performs a basic health check of all services
func healthCheck(cfg *config.Config) error {
	fmt.Println("Performing health check...")

	// Check database connection
	db, err := database.NewEnhanced(&cfg.Database)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()
	fmt.Println("✓ Database connection OK")

	// Check Loki connectivity
	lokiClient := loki.New(&cfg.Loki)
	labels, err := lokiClient.GetLogLabels()
	if err != nil {
		return fmt.Errorf("loki connection failed: %w", err)
	}
	fmt.Printf("✓ Loki connection OK (found %d labels)\n", len(labels))

	// Check S3 connectivity
	s3Client, err := s3.New(&cfg.S3)
	if err != nil {
		return fmt.Errorf("s3 initialization failed: %w", err)
	}
	_ = s3Client
	fmt.Println("✓ S3 client initialized")

	// Check Sentry initialization
	sentryClient, err := sentry.New(&cfg.Sentry)
	if err != nil {
		return fmt.Errorf("sentry initialization failed: %w", err)
	}
	defer sentryClient.Close()
	fmt.Println("✓ Sentry client initialized")

	// Check Swagger client
	swaggerClient, err := swagger.New(&cfg.Swagger)
	if err != nil {
		return fmt.Errorf("swagger client initialization failed: %w", err)
	}
	_ = swaggerClient
	fmt.Println("✓ Swagger client initialized")

	// Check Simulator client
	simulatorClient := simulator.New()
	fmt.Println("✓ Simulator client initialized")

	// Perform a simple simulation test
	testReq := &simulator.Request{
		Method: "GET",
		URL:    "https://httpbin.org/get",
	}
	_, err = simulatorClient.Simulate(testReq)
	if err != nil {
		return fmt.Errorf("simulator test failed: %w", err)
	}
	fmt.Println("✓ Simulator test OK")

	return nil
}
