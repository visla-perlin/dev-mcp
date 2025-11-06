package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dev-mcp/internal/config"
	"dev-mcp/internal/database"
	"dev-mcp/internal/llm"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/mcp/server"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

func main() {
	// Load configuration
	cfg, err := config.Load("./configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("Dev MCP initialized successfully!")
	fmt.Printf("Server listening on %s:%d\n", cfg.Server.Host, cfg.Server.Port)

	// Initialize database client
	var db *database.DB
	db, err = database.New(&cfg.Database)
	if err != nil {
		log.Printf("Warning: Failed to initialize database: %v", err)
	} else {
		defer db.Close()
		fmt.Println("Database client initialized")
	}

	// Initialize Loki client
	lokiClient := loki.New(&cfg.Loki)
	_ = lokiClient
	fmt.Println("Loki client initialized")

	// Initialize S3 client
	var s3Client *s3.Client
	s3Client, err = s3.New(&cfg.S3)
	if err != nil {
		log.Printf("Warning: Failed to initialize S3 client: %v", err)
	} else {
		_ = s3Client
		fmt.Println("S3 client initialized")
	}

	// Initialize Sentry client
	var sentryClient *sentry.Client
	sentryClient, err = sentry.New(&cfg.Sentry)
	if err != nil {
		log.Printf("Warning: Failed to initialize Sentry client: %v", err)
	} else {
		defer sentryClient.Close()
		fmt.Println("Sentry client initialized")
	}

	// Initialize Swagger client
	var swaggerClient *swagger.Client
	swaggerClient, err = swagger.New(&cfg.Swagger)
	if err != nil {
		log.Printf("Warning: Failed to initialize Swagger client: %v", err)
	} else {
		_ = swaggerClient
		fmt.Println("Swagger client initialized")
	}

	// Initialize LLM service
	var llmService *llm.Service
	llmService, err = llm.NewService(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize LLM service: %v", err)
	} else {
		defer llmService.Close()
		fmt.Println("LLM service initialized")

		// Perform health check
		if err := llmService.HealthCheck(nil); err != nil {
			log.Printf("Warning: LLM service health check failed: %v", err)
		} else {
			fmt.Println("LLM service health check passed")
		}
	}

	// Initialize simulator client
	simulatorClient := simulator.New()
	fmt.Println("Simulator client initialized")

	// Check if we should start in MCP mode
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		// Start MCP server using official SDK
		mcpServer := server.NewMCPServer(
			db,
			lokiClient,
			s3Client,
			sentryClient,
			swaggerClient,
			llmService,
			simulatorClient,
		)

		fmt.Println("Starting MCP server with official SDK...")

		// Create a context for graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Handle graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Start server in goroutine
		serverErrCh := make(chan error, 1)
		go func() {
			if err := mcpServer.Start(ctx); err != nil {
				serverErrCh <- err
			}
		}()

		// Wait for shutdown signal or server error
		select {
		case err := <-serverErrCh:
			log.Fatalf("MCP server error: %v", err)
		case sig := <-sigCh:
			fmt.Printf("\nReceived signal %v, shutting down gracefully...\n", sig)
			cancel()
			mcpServer.Close()
		}

		return
	}

	// Example usage of the clients
	fmt.Println("\n--- Example Usage ---")

	// Example: Database query (only if connected)
	if db != nil {
		fmt.Println("Database connected - ready to query tables and data")
	}

	// Example: Loki query
	fmt.Println("Loki client ready - can query logs with LogQL")

	// Example: S3 access (only if configured)
	if s3Client != nil {
		fmt.Println("S3 client ready - can access JSON data from S3 URLs")
	}

	// Example: Sentry integration
	if sentryClient != nil {
		sentryClient.CaptureMessage("Dev MCP started successfully", "info", map[string]string{
			"component": "main",
		})
		fmt.Println("Sentry integration ready - can capture exceptions and messages")
	}

	// Example: Swagger parsing (only if configured)
	if swaggerClient != nil {
		fmt.Println("Swagger client ready - can parse API specifications")
	}

	// Example: LLM service (only if configured)
	if llmService != nil {
		fmt.Println("LLM service ready - can process natural language queries")

		// List available models
		models, err := llmService.ListModels(nil)
		if err != nil {
			log.Printf("Warning: Failed to list LLM models: %v", err)
		} else {
			fmt.Printf("Available LLM models: %v\n", models)
		}
	}

	// Example: Request simulation
	simReq := &simulator.Request{
		Method: "GET",
		URL:    "https://httpbin.org/get",
		Headers: map[string]string{
			"User-Agent": "Dev-MCP/1.0",
		},
		Timeout: 10,
	}

	resp, err := simulatorClient.Simulate(simReq)
	if err != nil {
		log.Printf("Warning: Failed to simulate request: %v", err)
	} else {
		fmt.Printf("Simulated request successful - Status: %d, Time: %v\n", resp.StatusCode, resp.TimeTaken)
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

	// Keep the program running
	select {}
}

// healthCheck performs a basic health check of all services
func healthCheck(cfg *config.Config) error {
	fmt.Println("Performing health check...")

	// Check database connection
	db, err := database.New(&cfg.Database)
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