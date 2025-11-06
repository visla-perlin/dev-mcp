package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"dev-mcp/internal/config"
	"dev-mcp/internal/logging"

	_ "github.com/go-sql-driver/mysql"
)

// EnhancedDB represents an enhanced database connection with reconnection capabilities
type EnhancedDB struct {
	config          *config.DatabaseConfig
	db              *sql.DB
	mu              sync.RWMutex
	connected       bool
	lastError       error
	reconnectTicker *time.Ticker
	stopReconnect   chan bool
	logger          *logging.Logger
	sqlValidator    *SQLValidator
}

// ConnectionStatus represents the current connection status
type ConnectionStatus struct {
	Connected    bool      `json:"connected"`
	LastPing     time.Time `json:"last_ping"`
	LastError    string    `json:"last_error"`
	Reconnecting bool      `json:"reconnecting"`
	AttemptCount int       `json:"attempt_count"`
}

// NewEnhanced creates a new enhanced database connection with reconnection capabilities
func NewEnhanced(cfg *config.DatabaseConfig) (*EnhancedDB, error) {
	logger := logging.New("Database")

	edb := &EnhancedDB{
		config:        cfg,
		stopReconnect: make(chan bool, 1),
		logger:        logger,
		sqlValidator:  NewSQLValidator(), // Initialize with secure defaults
	}

	// Validate configuration first
	if err := edb.validateConfig(); err != nil {
		return nil, fmt.Errorf("database configuration validation failed: %w", err)
	}

	// Try initial connection
	if err := edb.connect(); err != nil {
		logger.Error("initial database connection failed",
			logging.String("error", err.Error()))
		// Start reconnection routine
		edb.startReconnectRoutine()
		return edb, nil // Return the object even if initial connection failed
	}

	logger.Info("database connection established successfully")
	return edb, nil
}

// validateConfig validates the database configuration
func (edb *EnhancedDB) validateConfig() error {
	missing := []string{}

	if edb.config.Host == "" {
		missing = append(missing, "host")
	}
	if edb.config.Port == 0 {
		missing = append(missing, "port")
	}
	if edb.config.Username == "" {
		missing = append(missing, "username")
	}
	if edb.config.Password == "" {
		missing = append(missing, "password")
	}
	if edb.config.DBName == "" {
		missing = append(missing, "dbname")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration fields: %v", missing)
	}

	return nil
}

// connect establishes the database connection
func (edb *EnhancedDB) connect() error {
	edb.mu.Lock()
	defer edb.mu.Unlock()

	// Close existing connection if any
	if edb.db != nil {
		edb.db.Close()
	}

	// MySQL connection string format
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		edb.config.Username, edb.config.Password,
		edb.config.Host, edb.config.Port, edb.config.DBName)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		edb.lastError = err
		edb.connected = false
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		edb.lastError = err
		edb.connected = false
		return fmt.Errorf("failed to ping database: %w", err)
	}

	edb.db = db
	edb.connected = true
	edb.lastError = nil

	return nil
}

// startReconnectRoutine starts the automatic reconnection routine
func (edb *EnhancedDB) startReconnectRoutine() {
	edb.reconnectTicker = time.NewTicker(30 * time.Second)

	go func() {
		defer edb.reconnectTicker.Stop()

		attemptCount := 0
		for {
			select {
			case <-edb.reconnectTicker.C:
				if !edb.IsConnected() {
					attemptCount++
					edb.logger.Info("attempting database reconnection",
						logging.String("attempt", fmt.Sprintf("%d", attemptCount)))

					if err := edb.connect(); err != nil {
						edb.logger.Error("reconnection failed",
							logging.String("error", err.Error()),
							logging.String("attempt", fmt.Sprintf("%d", attemptCount)))
					} else {
						edb.logger.Info("database reconnection successful")
						attemptCount = 0
					}
				}
			case <-edb.stopReconnect:
				return
			}
		}
	}()
}

// IsConnected returns whether the database is currently connected
func (edb *EnhancedDB) IsConnected() bool {
	edb.mu.RLock()
	defer edb.mu.RUnlock()
	return edb.connected
}

// GetStatus returns the current connection status
func (edb *EnhancedDB) GetStatus() ConnectionStatus {
	edb.mu.RLock()
	defer edb.mu.RUnlock()

	status := ConnectionStatus{
		Connected:    edb.connected,
		Reconnecting: edb.reconnectTicker != nil,
	}

	if edb.lastError != nil {
		status.LastError = edb.lastError.Error()
	}

	return status
}

// HealthCheck performs a health check on the database connection
func (edb *EnhancedDB) HealthCheck() error {
	if !edb.IsConnected() {
		return fmt.Errorf("database not connected")
	}

	edb.mu.RLock()
	db := edb.db
	edb.mu.RUnlock()

	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		edb.mu.Lock()
		edb.connected = false
		edb.lastError = err
		edb.mu.Unlock()
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// ExecuteQuery executes a query with connection checking and security validation
func (edb *EnhancedDB) ExecuteQuery(query string, args ...interface{}) (*sql.Rows, error) {
	// First, validate the SQL for security
	if err := edb.sqlValidator.ValidateSQL(query); err != nil {
		edb.logger.Warn("SQL security validation failed",
			logging.String("query", query),
			logging.String("error", err.Error()))
		return nil, fmt.Errorf("SQL security validation failed: %w", err)
	}

	if !edb.IsConnected() {
		return nil, fmt.Errorf("database not connected - query: %s", query)
	}

	edb.mu.RLock()
	db := edb.db
	edb.mu.RUnlock()

	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	edb.logger.Info("executing validated SQL query",
		logging.String("query", query))

	return db.Query(query, args...)
}

// GetTables returns a list of all tables in the database
func (edb *EnhancedDB) GetTables() ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE()
		ORDER BY table_name
	`

	rows, err := edb.ExecuteQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over table rows: %w", err)
	}

	return tables, nil
}

// Close closes the database connection and stops reconnection
func (edb *EnhancedDB) Close() error {
	// Stop reconnection routine
	if edb.reconnectTicker != nil {
		edb.stopReconnect <- true
		edb.reconnectTicker.Stop()
	}

	edb.mu.Lock()
	defer edb.mu.Unlock()

	if edb.db != nil {
		err := edb.db.Close()
		edb.db = nil
		edb.connected = false
		return err
	}

	return nil
}

// GetDB returns the underlying database connection
func (edb *EnhancedDB) GetDB() *sql.DB {
	edb.mu.RLock()
	defer edb.mu.RUnlock()
	return edb.db
}

// GetSQLValidator returns the SQL validator instance
func (edb *EnhancedDB) GetSQLValidator() *SQLValidator {
	return edb.sqlValidator
}

// SetSQLValidator sets a custom SQL validator
func (edb *EnhancedDB) SetSQLValidator(validator *SQLValidator) {
	edb.sqlValidator = validator
}

// EnableUnsafeMode enables unsafe SQL operations (use with extreme caution)
func (edb *EnhancedDB) EnableUnsafeMode() {
	edb.logger.Warn("enabling unsafe SQL mode - all operations will be allowed")
	edb.sqlValidator = NewUnsafeSQLValidator()
}

// DisableUnsafeMode disables unsafe SQL operations and returns to secure defaults
func (edb *EnhancedDB) DisableUnsafeMode() {
	edb.logger.Info("disabling unsafe SQL mode - returning to secure defaults")
	edb.sqlValidator = NewSQLValidator()
}

// IsUnsafeModeEnabled returns whether unsafe mode is currently enabled
func (edb *EnhancedDB) IsUnsafeModeEnabled() bool {
	return edb.sqlValidator.GetPolicy().AllowUnsafeMode
}

// GetAllowedOperations returns the list of currently allowed SQL operations
func (edb *EnhancedDB) GetAllowedOperations() []string {
	if edb.sqlValidator.GetPolicy().AllowUnsafeMode {
		return []string{"ALL_OPERATIONS"}
	}
	return edb.sqlValidator.GetPolicy().AllowedOperations
}

// GetBlockedOperations returns the list of currently blocked SQL operations
func (edb *EnhancedDB) GetBlockedOperations() []string {
	if edb.sqlValidator.GetPolicy().AllowUnsafeMode {
		return []string{}
	}
	return edb.sqlValidator.GetPolicy().BlockedOperations
}
