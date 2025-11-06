package database

import (
	"fmt"
	"regexp"
	"strings"
)

// SQLSecurityPolicy defines the security policy for SQL operations
type SQLSecurityPolicy struct {
	AllowedOperations []string
	BlockedOperations []string
	AllowUnsafeMode   bool
}

// SQLValidator provides SQL security validation
type SQLValidator struct {
	policy *SQLSecurityPolicy
}

// NewSQLValidator creates a new SQL validator with default security policy
func NewSQLValidator() *SQLValidator {
	return &SQLValidator{
		policy: &SQLSecurityPolicy{
			AllowedOperations: []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN"},
			BlockedOperations: []string{"DELETE", "DROP", "UPDATE", "TRUNCATE", "INSERT", "ALTER", "CREATE", "GRANT", "REVOKE"},
			AllowUnsafeMode:   false,
		},
	}
}

// NewUnsafeSQLValidator creates a validator that allows all operations (use with caution)
func NewUnsafeSQLValidator() *SQLValidator {
	return &SQLValidator{
		policy: &SQLSecurityPolicy{
			AllowedOperations: []string{}, // Empty means allow all when unsafe mode is on
			BlockedOperations: []string{},
			AllowUnsafeMode:   true,
		},
	}
}

// ValidateSQL validates if the SQL query is safe to execute
func (v *SQLValidator) ValidateSQL(query string) error {
	if v.policy.AllowUnsafeMode {
		return nil // Allow everything in unsafe mode
	}

	// Clean and normalize the query
	cleanQuery := strings.TrimSpace(strings.ToUpper(query))
	if cleanQuery == "" {
		return fmt.Errorf("empty query not allowed")
	}

	// Extract the main operation (first significant word)
	operation := v.extractOperation(cleanQuery)

	// Check if operation is explicitly blocked
	for _, blockedOp := range v.policy.BlockedOperations {
		if operation == strings.ToUpper(blockedOp) {
			return fmt.Errorf("operation '%s' is blocked for security reasons. Only read-only operations are allowed: %s",
				operation, strings.Join(v.policy.AllowedOperations, ", "))
		}
	}

	// Check if operation is in allowed list (if allowed list is not empty)
	if len(v.policy.AllowedOperations) > 0 {
		allowed := false
		for _, allowedOp := range v.policy.AllowedOperations {
			if operation == strings.ToUpper(allowedOp) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("operation '%s' is not allowed. Allowed operations: %s",
				operation, strings.Join(v.policy.AllowedOperations, ", "))
		}
	}

	// Additional security checks
	if err := v.checkForDangerousPatterns(cleanQuery); err != nil {
		return err
	}

	return nil
}

// extractOperation extracts the main SQL operation from the query
func (v *SQLValidator) extractOperation(query string) string {
	// Remove comments
	query = v.removeComments(query)

	// Split by whitespace and get first non-empty word
	words := strings.Fields(query)
	if len(words) == 0 {
		return ""
	}

	return words[0]
}

// removeComments removes SQL comments from the query
func (v *SQLValidator) removeComments(query string) string {
	// Remove single line comments (-- comment)
	singleLineComment := regexp.MustCompile(`--.*$`)
	query = singleLineComment.ReplaceAllString(query, "")

	// Remove multi-line comments (/* comment */)
	multiLineComment := regexp.MustCompile(`/\*.*?\*/`)
	query = multiLineComment.ReplaceAllString(query, "")

	return query
}

// checkForDangerousPatterns checks for dangerous SQL patterns
func (v *SQLValidator) checkForDangerousPatterns(query string) error {
	dangerousPatterns := []struct {
		pattern     *regexp.Regexp
		description string
	}{
		{regexp.MustCompile(`;\s*(DELETE|DROP|UPDATE|TRUNCATE|INSERT|ALTER|CREATE)`), "multiple statements with dangerous operations"},
		{regexp.MustCompile(`UNION.*?(DELETE|DROP|UPDATE|TRUNCATE|INSERT|ALTER|CREATE)`), "UNION with dangerous operations"},
		{regexp.MustCompile(`(EXEC|EXECUTE|SP_|XP_)`), "stored procedure execution"},
		{regexp.MustCompile(`(LOAD_FILE|INTO\s+OUTFILE|INTO\s+DUMPFILE)`), "file system operations"},
		{regexp.MustCompile(`(BENCHMARK|SLEEP)`), "timing attack functions"},
	}

	for _, pattern := range dangerousPatterns {
		if pattern.pattern.MatchString(query) {
			return fmt.Errorf("dangerous SQL pattern detected: %s", pattern.description)
		}
	}

	return nil
}

// GetPolicy returns the current security policy
func (v *SQLValidator) GetPolicy() *SQLSecurityPolicy {
	return v.policy
}

// SetPolicy sets a new security policy
func (v *SQLValidator) SetPolicy(policy *SQLSecurityPolicy) {
	v.policy = policy
}

// IsOperationAllowed checks if a specific operation is allowed
func (v *SQLValidator) IsOperationAllowed(operation string) bool {
	if v.policy.AllowUnsafeMode {
		return true
	}

	operation = strings.ToUpper(operation)

	// Check if explicitly blocked
	for _, blockedOp := range v.policy.BlockedOperations {
		if operation == strings.ToUpper(blockedOp) {
			return false
		}
	}

	// If there's an allowed list, check it
	if len(v.policy.AllowedOperations) > 0 {
		for _, allowedOp := range v.policy.AllowedOperations {
			if operation == strings.ToUpper(allowedOp) {
				return true
			}
		}
		return false
	}

	return true
}
