package tools

import (
	"dev-mcp/internal/config"
	"dev-mcp/internal/logging"
	"dev-mcp/internal/loki"
	"dev-mcp/internal/s3"
	"dev-mcp/internal/sentry"
	"dev-mcp/internal/simulator"
	"dev-mcp/internal/swagger"
)

// ToolRegistry manages the registration of all tools
type ToolRegistry struct {
	logger         *logging.Logger
	serviceManager *config.ServiceManager
	registrar      ToolRegistrar
}

// ToolContext contains all the services that tools might need
type ToolContext struct {
	Config          *config.Config
	DatabaseManager *DatabaseManager
	LokiClient      *loki.Client
	S3Client        *s3.Client
	SentryClient    *sentry.Client
	SwaggerClient   *swagger.Client
	SimulatorClient *simulator.Client
	ServiceManager  *config.ServiceManager
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry(registrar ToolRegistrar, serviceManager *config.ServiceManager) *ToolRegistry {
	return &ToolRegistry{
		logger:         logging.New("ToolRegistry"),
		serviceManager: serviceManager,
		registrar:      registrar,
	}
}

// RegisterAll registers all available tools based on the provided context
func (tr *ToolRegistry) RegisterAll(ctx *ToolContext) {
	tr.logger.Info("Starting tool registration process...")

	// Register file tools (always available)
	tr.registerFileTools()

	// Register database tools if database manager is available
	if ctx.DatabaseManager != nil {
		ctx.DatabaseManager.RegisterDatabaseTools(tr.registrar)
	}

	// Register other service tools if available
	if ctx.LokiClient != nil {
		tr.registerLokiTools(ctx.LokiClient)
	}

	if ctx.S3Client != nil {
		tr.registerS3Tools(ctx.S3Client)
	}

	if ctx.SentryClient != nil {
		tr.registerSentryTools(ctx.SentryClient)
	}

	if ctx.SwaggerClient != nil {
		tr.registerSwaggerTools(ctx.SwaggerClient)
	}

	if ctx.SimulatorClient != nil {
		tr.registerSimulatorTools(ctx.SimulatorClient)
	}

	tr.logger.Info("Tool registration completed successfully")
}

// registerFileTools registers all file management tools
func (tr *ToolRegistry) registerFileTools() {
	tr.logger.Info("Registering file tools...")
	RegisterFileTools(tr.registrar)
}

// registerLokiTools registers Loki-related tools
func (tr *ToolRegistry) registerLokiTools(lokiClient *loki.Client) {
	tr.logger.Info("Registering Loki tools...")
	tr.registrar.RegisterTool(NewLokiQueryTool(lokiClient))
	tr.logger.Info("Registered tool", logging.String("name", "loki_query"))
}

// registerS3Tools registers S3-related tools
func (tr *ToolRegistry) registerS3Tools(s3Client *s3.Client) {
	tr.logger.Info("Registering S3 tools...")
	tr.registrar.RegisterTool(NewS3QueryTool(s3Client))
	tr.logger.Info("Registered tool", logging.String("name", "s3_query"))
}

// registerSentryTools registers Sentry-related tools
func (tr *ToolRegistry) registerSentryTools(sentryClient *sentry.Client) {
	tr.logger.Info("Registering Sentry tools...")
	tr.registrar.RegisterTool(NewSentryQueryTool(sentryClient))
	tr.logger.Info("Registered tool", logging.String("name", "sentry_query"))
}

// registerSwaggerTools registers Swagger-related tools
func (tr *ToolRegistry) registerSwaggerTools(swaggerClient *swagger.Client) {
	tr.logger.Info("Registering Swagger tools...")
	tr.registrar.RegisterTool(NewSwaggerQueryTool(swaggerClient))
	tr.logger.Info("Registered tool", logging.String("name", "swagger_query"))
}

// registerSimulatorTools registers simulator-related tools
func (tr *ToolRegistry) registerSimulatorTools(simulatorClient *simulator.Client) {
	tr.logger.Info("Registering Simulator tools...")
	tr.registrar.RegisterTool(NewSimulatorTool(simulatorClient))
	tr.logger.Info("Registered tool", logging.String("name", "simulator"))
}
