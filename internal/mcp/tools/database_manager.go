package tools

import (
	"fmt"

	"dev-mcp/internal/config"
	"dev-mcp/internal/database"
	"dev-mcp/internal/logging"
)

// DatabaseManager manages database connections and provides database tools
type DatabaseManager struct {
	db             database.DatabaseInterface
	serviceManager *config.ServiceManager
	logger         *logging.Logger
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(cfg *config.DatabaseConfig, serviceManager *config.ServiceManager) *DatabaseManager {
	logger := logging.New("DatabaseManager")

	// Try to create enhanced database connection
	db, err := database.NewEnhanced(cfg)
	if err != nil {
		logger.Error("failed to create database connection", logging.String("error", err.Error()))
		// Return manager with nil db - tools will handle this gracefully
		return &DatabaseManager{
			db:             nil,
			serviceManager: serviceManager,
			logger:         logger,
		}
	}

	return &DatabaseManager{
		db:             db,
		serviceManager: serviceManager,
		logger:         logger,
	}
}

// IsAvailable returns whether the database is available
func (dm *DatabaseManager) IsAvailable() bool {
	return dm.db != nil && dm.db.IsConnected()
}

// GetDatabase returns the database interface (for internal use only)
func (dm *DatabaseManager) GetDatabase() database.DatabaseInterface {
	return dm.db
}

// RegisterDatabaseTools registers all database-related tools
func (dm *DatabaseManager) RegisterDatabaseTools(registrar ToolRegistrar) {
	if dm.db == nil {
		dm.logger.Warn("database not available, skipping database tools registration")
		return
	}

	dm.logger.Info("registering database tools...")

	// Register unified database tool
	registrar.RegisterTool(*NewDatabaseTool(dm.db, dm.serviceManager))
	dm.logger.Info("registered tool", logging.String("name", "database_query"))

	// Register database security tool
	registrar.RegisterTool(*NewDatabaseSecurityTool(dm.db, dm.serviceManager))
	dm.logger.Info("registered tool", logging.String("name", "database_security"))
}

// Close closes the database connection
func (dm *DatabaseManager) Close() error {
	if dm.db != nil {
		return dm.db.Close()
	}
	return nil
}

// Health check for the database
func (dm *DatabaseManager) HealthCheck() error {
	if dm.db == nil {
		return fmt.Errorf("database not available")
	}
	return dm.db.HealthCheck()
}
