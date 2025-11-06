package database

import (
	"database/sql"
	"fmt"

	"dev-mcp/internal/config"
	_ "github.com/go-sql-driver/mysql"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(cfg *config.DatabaseConfig) (*DB, error) {
	// MySQL connection string format
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// GetTableSchema returns the schema of a table
func (db *DB) GetTableSchema(tableName string) ([]map[string]interface{}, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = ? AND table_schema = DATABASE()
		ORDER BY ordinal_position
	`

	rows, err := db.DB.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query table schema: %w", err)
	}
	defer rows.Close()

	var schema []map[string]interface{}
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString

		err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		columnInfo := map[string]interface{}{
			"column_name":    columnName,
			"data_type":      dataType,
			"is_nullable":    isNullable,
			"column_default": columnDefault,
		}

		schema = append(schema, columnInfo)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return schema, nil
}

// Query executes a SQL query and returns the results
func (db *DB) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
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
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map to hold the row data
		rowData := make(map[string]interface{})
		for i, col := range columns {
			rowData[col] = values[i]
		}

		results = append(results, rowData)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}