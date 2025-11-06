package server

import (
	"log"

	"dev-mcp/internal/config"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/mcp/tools"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

// ServiceContainer holds all initialized services
type ServiceContainer struct {
	DatabaseManager *tools.DatabaseManager
	LokiClient      *loki.Client
	S3Client        *s3.Client
	SentryClient    *sentry.Client
	SwaggerClient   *swagger.Client
	SimulatorClient *simulator.Client
}

// InitializeServices initializes all services based on the configuration
func InitializeServices(cfg *config.Config) *ServiceContainer {
	log.Println("Initializing Dev MCP Server components...")

	container := &ServiceContainer{}

	// Create service manager first
	serviceManager := config.NewServiceManager(cfg)

	// Initialize database manager (encapsulates database logic)
	container.DatabaseManager = tools.NewDatabaseManager(&cfg.Database, serviceManager)
	if container.DatabaseManager.IsAvailable() {
		log.Println("✓ Database manager initialized")
	} else {
		log.Println("⚠ Database manager initialized (connection failed)")
	}

	// Initialize Loki client
	container.LokiClient = loki.New(&cfg.Loki)
	log.Println("✓ Loki client initialized")

	// Initialize S3 client
	if s3Client, err := s3.New(&cfg.S3); err != nil {
		log.Printf("Warning: Failed to initialize S3 client: %v", err)
	} else {
		container.S3Client = s3Client
		log.Println("✓ S3 client initialized")
	}

	// Initialize Sentry client
	if sentryClient, err := sentry.New(&cfg.Sentry); err != nil {
		log.Printf("Warning: Failed to initialize Sentry client: %v", err)
	} else {
		container.SentryClient = sentryClient
		log.Println("✓ Sentry client initialized")
	}

	// Initialize Swagger client
	if swaggerClient, err := swagger.New(&cfg.Swagger); err != nil {
		log.Printf("Warning: Failed to initialize Swagger client: %v", err)
	} else {
		container.SwaggerClient = swaggerClient
		log.Println("✓ Swagger client initialized")
	}

	// Initialize simulator client
	container.SimulatorClient = simulator.New()
	log.Println("✓ Simulator client initialized")

	return container
}

// Close closes all services that need cleanup
func (sc *ServiceContainer) Close() {
	if sc.DatabaseManager != nil {
		sc.DatabaseManager.Close()
	}
	if sc.SentryClient != nil {
		sc.SentryClient.Close()
	}
}
