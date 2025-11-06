package main

import (
	"fmt"
	"log"

	"dev-mcp/internal/config"
	"dev-mcp/internal/database"
)

func main() {
	fmt.Println("🔒 Testing SQL Security Validator")
	fmt.Println("================================")

	// Create a test database config
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "test",
		Password: "test",
		DBName:   "test",
		SSLMode:  "disable",
	}

	// Create enhanced database (won't actually connect)
	db, err := database.NewEnhanced(cfg)
	if err != nil {
		log.Fatalf("Failed to create enhanced database: %v", err)
	}

	// Get the SQL validator
	validator := db.GetSQLValidator()

	// Test queries
	testQueries := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM products WHERE price > 100",
		"SHOW TABLES",
		"DESCRIBE users",
		"EXPLAIN SELECT * FROM orders",
		"DELETE FROM users WHERE id = 1",
		"DROP TABLE users",
		"UPDATE users SET name = 'John' WHERE id = 1",
		"TRUNCATE TABLE logs",
		"INSERT INTO users (name) VALUES ('Alice')",
		"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
		"CREATE TABLE test (id INT)",
		"SELECT * FROM users; DROP TABLE users;",
		"SELECT * FROM users UNION SELECT * FROM admin_users",
		"SELECT BENCHMARK(1000000, SHA1('test'))",
		"SELECT LOAD_FILE('/etc/passwd')",
	}

	fmt.Println("\n🔍 Testing Queries:")
	fmt.Println("------------------")

	for i, query := range testQueries {
		fmt.Printf("\n%d. Query: %s\n", i+1, query)

		err := validator.ValidateSQL(query)
		if err != nil {
			fmt.Printf("   ❌ BLOCKED: %s\n", err.Error())
		} else {
			fmt.Printf("   ✅ ALLOWED\n")
		}
	}

	// Test unsafe mode
	fmt.Println("\n⚠️  Testing Unsafe Mode:")
	fmt.Println("------------------------")

	db.EnableUnsafeMode()
	fmt.Printf("Unsafe mode enabled: %t\n", db.IsUnsafeModeEnabled())

	// Test same dangerous queries in unsafe mode
	dangerousQueries := []string{
		"DELETE FROM users WHERE id = 1",
		"DROP TABLE users",
		"UPDATE users SET name = 'John'",
	}

	for _, query := range dangerousQueries {
		fmt.Printf("\nQuery: %s\n", query)
		// Get the updated validator after enabling unsafe mode
		updatedValidator := db.GetSQLValidator()
		err := updatedValidator.ValidateSQL(query)
		if err != nil {
			fmt.Printf("   ❌ BLOCKED: %s\n", err.Error())
		} else {
			fmt.Printf("   ✅ ALLOWED (UNSAFE MODE)\n")
		}
	}

	// Return to safe mode
	db.DisableUnsafeMode()
	fmt.Printf("\nUnsafe mode disabled: %t\n", !db.IsUnsafeModeEnabled())

	// Show security policy
	fmt.Println("\n🛡️  Security Policy:")
	fmt.Println("-------------------")
	fmt.Printf("Allowed operations: %v\n", db.GetAllowedOperations())
	fmt.Printf("Blocked operations: %v\n", db.GetBlockedOperations())

	fmt.Println("\n✅ SQL Security Test Complete!")
}
