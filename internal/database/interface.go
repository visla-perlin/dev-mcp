package database

import (
	"database/sql"
)

// DatabaseInterface defines the common interface for both DB and EnhancedDB
type DatabaseInterface interface {
	// Core database operations
	GetTables() ([]string, error)
	GetTableSchema(tableName string) ([]map[string]interface{}, error)
	Query(query string, args ...interface{}) ([]map[string]interface{}, error)
	Close() error

	// Health and status operations
	HealthCheck() error
	IsConnected() bool

	// Get underlying connection for legacy compatibility
	GetUnderlyingDB() *sql.DB
}

// Ensure DB implements DatabaseInterface
var _ DatabaseInterface = (*DB)(nil)

// Ensure EnhancedDB implements DatabaseInterface
var _ DatabaseInterface = (*EnhancedDB)(nil)

// GetUnderlyingDB returns the underlying sql.DB for legacy DB
func (db *DB) GetUnderlyingDB() *sql.DB {
	return db.DB
}

// HealthCheck performs a simple ping for legacy DB
func (db *DB) HealthCheck() error {
	return db.Ping()
}

// IsConnected checks if the legacy DB is connected
func (db *DB) IsConnected() bool {
	return db.Ping() == nil
}

// GetUnderlyingDB returns the underlying sql.DB for EnhancedDB
func (edb *EnhancedDB) GetUnderlyingDB() *sql.DB {
	return edb.GetDB()
}

// Query implementation for EnhancedDB that uses secure validation
func (edb *EnhancedDB) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := edb.ExecuteQuery(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

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
			return nil, err
		}

		// Create a map to hold the row data
		rowData := make(map[string]interface{})
		for i, col := range columns {
			rowData[col] = values[i]
		}

		results = append(results, rowData)
	}

	return results, rows.Err()
}

// GetTableSchema implementation for EnhancedDB
func (edb *EnhancedDB) GetTableSchema(tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = ? AND table_schema = DATABASE()
		ORDER BY ordinal_position
	`

	return edb.Query(query, tableName)
}
