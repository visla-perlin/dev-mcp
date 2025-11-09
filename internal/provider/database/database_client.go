package database

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"dev-mcp/internal/config"
	"dev-mcp/internal/logging"
)

// DatabaseClient provides secure database operations
type DatabaseClient struct {
	db         *sql.DB
	config     *config.DatabaseConfig
	logger     *logging.Logger
	unsafeMode bool
	allowedOps []string
	blockedOps []string
	mu         sync.RWMutex
}

// IsAvailable checks if Database client is available
func (c *DatabaseClient) IsAvailable() bool {
	return c.db != nil && c.db.Ping() == nil
}

// NewDatabaseClient creates a new secure database client
func NewDatabaseClient(cfg *config.DatabaseConfig) (*DatabaseClient, error) {
	logger := logging.New("DatabaseClient")

	if cfg == nil || cfg.Host == "" || cfg.Username == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("database configuration is incomplete")
	}

	// For now, we'll assume MySQL since that's what the config supports
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		logger.Error("failed to open database connection", logging.String("error", err.Error()))
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		logger.Error("failed to ping database", logging.String("error", err.Error()))
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	client := &DatabaseClient{
		db:         db,
		config:     cfg,
		logger:     logger,
		unsafeMode: false,
		allowedOps: []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN"},
		blockedOps: []string{"INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE", "ALTER", "CREATE"},
	}

	logger.Info("database client initialized successfully")
	return client, nil
}

// Query executes a secure SQL query with validation
func (c *DatabaseClient) Query(query string) ([]map[string]interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Validate the query for security
	if err := c.validateQuery(query); err != nil {
		return nil, fmt.Errorf("SQL security validation failed: %w", err)
	}

	// Execute the query
	rows, err := c.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Read all rows
	var results []map[string]interface{}
	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Handle []byte (common for strings in some drivers)
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return results, nil
}

// validateQuery performs security validation on SQL queries
func (c *DatabaseClient) validateQuery(query string) error {
	// Trim whitespace
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("empty query")
	}

	// Extract the first word (operation type)
	re := regexp.MustCompile(`^\s*(\w+)`)
	matches := re.FindStringSubmatch(query)
	if len(matches) < 2 {
		return fmt.Errorf("invalid query format")
	}

	operation := strings.ToUpper(matches[1])

	// Check if unsafe mode is enabled
	if c.unsafeMode {
		if c.logger != nil {
			c.logger.Warn("unsafe mode enabled - bypassing security checks", logging.String("operation", operation))
		}
		return nil
	}

	// First, check for dangerous patterns before allowing any operations
	if c.hasDangerousPatterns(query) {
		return fmt.Errorf("query contains potentially dangerous patterns")
	}

	// Check allowed operations
	allowedOps := c.allowedOps
	if allowedOps == nil {
		// Use default allowed operations if not set
		allowedOps = []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN"}
	}
	for _, allowed := range allowedOps {
		if operation == allowed {
			if c.logger != nil {
				c.logger.Debug("allowed operation", logging.String("operation", operation))
			}
			return nil
		}
	}

	// Check blocked operations
	blockedOps := c.blockedOps
	if blockedOps == nil {
		// Use default blocked operations if not set
		blockedOps = []string{"INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE", "ALTER", "CREATE"}
	}
	for _, blocked := range blockedOps {
		if operation == blocked {
			return fmt.Errorf("operation '%s' is blocked for security reasons", operation)
		}
	}

	// Default to allowed if no explicit rules match
	if c.logger != nil {
		c.logger.Debug("operation allowed by default", logging.String("operation", operation))
	}
	return nil
}

// hasDangerousPatterns checks for potentially dangerous SQL patterns
func (c *DatabaseClient) hasDangerousPatterns(query string) bool {
	query = strings.ToUpper(query)

	// Check for comment-based SQL injection
	if strings.Contains(query, "/*") || strings.Contains(query, "--") {
		return true
	}

	// Check for stacked queries (any combination of statements separated by semicolons)
	if strings.Contains(query, ";") {
		// Split the query by semicolons and check each part
		parts := strings.Split(query, ";")
		if len(parts) > 1 {
			// More than one statement - check each one
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					// Extract operation from each part
					re := regexp.MustCompile(`^\s*(\w+)`)
					matches := re.FindStringSubmatch(part)
					if len(matches) >= 2 {
						op := strings.ToUpper(matches[1])
						// Check if any part is a dangerous operation
						blockedOps := c.blockedOps
						if blockedOps == nil {
							blockedOps = []string{"INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE", "ALTER", "CREATE"}
						}
						for _, blocked := range blockedOps {
							if op == blocked {
								return true
							}
						}
					}
				}
			}
		}
	}

	// Check for common SQL injection patterns
	dangerousPatterns := []string{
		"UNION.*SELECT",
		"DROP.*TABLE",
		"DELETE.*FROM",
		"INSERT.*INTO",
		"UPDATE.*SET",
		"TRUNCATE.*TABLE",
		"ALTER.*TABLE",
		"CREATE.*TABLE",
		"EXEC.*SP_",
		"EXECUTE.*SP_",
	}

	for _, pattern := range dangerousPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(query) {
			return true
		}
	}

	return false
}

// EnableUnsafeMode enables unsafe mode (allows all operations)
func (c *DatabaseClient) EnableUnsafeMode() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unsafeMode = true
	if c.logger != nil {
		c.logger.Warn("unsafe mode enabled")
	}
}

// DisableUnsafeMode disables unsafe mode (restricts to safe operations)
func (c *DatabaseClient) DisableUnsafeMode() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unsafeMode = false
	if c.logger != nil {
		c.logger.Info("unsafe mode disabled")
	}
}

// IsUnsafeModeEnabled returns whether unsafe mode is enabled
func (c *DatabaseClient) IsUnsafeModeEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.unsafeMode
}

// GetAllowedOperations returns the list of allowed operations
func (c *DatabaseClient) GetAllowedOperations() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification
	// If allowedOps is nil, return default allowed operations
	if c.allowedOps == nil {
		defaultAllowed := []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN"}
		allowed := make([]string, len(defaultAllowed))
		copy(allowed, defaultAllowed)
		return allowed
	}
	allowed := make([]string, len(c.allowedOps))
	copy(allowed, c.allowedOps)
	return allowed
}

// GetBlockedOperations returns the list of blocked operations
func (c *DatabaseClient) GetBlockedOperations() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification
	// If blockedOps is nil, return default blocked operations
	if c.blockedOps == nil {
		defaultBlocked := []string{"INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE", "ALTER", "CREATE"}
		blocked := make([]string, len(defaultBlocked))
		copy(blocked, defaultBlocked)
		return blocked
	}
	blocked := make([]string, len(c.blockedOps))
	copy(blocked, c.blockedOps)
	return blocked
}

// Close closes the database connection
func (c *DatabaseClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.logger.Error("failed to close database connection", logging.String("error", err.Error()))
			return err
		}
		c.logger.Info("database connection closed")
	}
	return nil
}

// HealthCheck performs a health check on the database connection
func (c *DatabaseClient) HealthCheck() error {
	if c.db == nil {
		return fmt.Errorf("database not initialized")
	}

	if err := c.db.Ping(); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	c.logger.Debug("database health check passed")
	return nil
}

// ValidateQueryForTest validates a query for testing purposes (exported for test access)
func (c *DatabaseClient) ValidateQueryForTest(query string) error {
	return c.validateQuery(query)
}