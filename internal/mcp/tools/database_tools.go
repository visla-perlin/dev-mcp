package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/config"
	"dev-mcp/internal/database"
	"dev-mcp/internal/logging"
)

// DatabaseTool represents a unified database tool with security validation
type DatabaseTool struct {
	db             database.DatabaseInterface
	serviceManager *config.ServiceManager
	logger         *logging.Logger
}

// NewDatabaseTool creates a new unified database tool
func NewDatabaseTool(db database.DatabaseInterface, serviceManager *config.ServiceManager) *ToolDefinition {
	logger := logging.New("DatabaseTool")

	tool := &DatabaseTool{
		db:             db,
		serviceManager: serviceManager,
		logger:         logger,
	}

	return &ToolDefinition{
		Tool: &mcp.Tool{
			Name:        "database_query",
			Description: "Execute secure database queries and manage database operations. Only read-only operations are allowed by default (SELECT, SHOW, DESCRIBE, EXPLAIN). Write operations are blocked for security unless unsafe mode is enabled.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "string",
						"description": "SQL query to execute (read-only operations only by default)"
					},
					"table": {
						"type": "string",
						"description": "Table name to query schema (optional, can be inferred from query)"
					}
				},
				"required": ["query"]
			}`),
		},
		Handler: tool.handleDatabaseQuery,
	}
}

// NewDatabaseSecurityTool creates a new database security management tool
func NewDatabaseSecurityTool(db database.DatabaseInterface, serviceManager *config.ServiceManager) *ToolDefinition {
	logger := logging.New("DatabaseSecurityTool")

	tool := &DatabaseTool{
		db:             db,
		serviceManager: serviceManager,
		logger:         logger,
	}

	return &ToolDefinition{
		Tool: &mcp.Tool{
			Name:        "database_security",
			Description: "Manage database security settings and view SQL operation policies. Requires admin role.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"action": {
						"type": "string",
						"description": "Action to perform: 'status', 'enable_unsafe', 'disable_unsafe', 'allowed_ops', 'blocked_ops'",
						"enum": ["status", "enable_unsafe", "disable_unsafe", "allowed_ops", "blocked_ops"]
					}
				},
				"required": ["action"]
			}`),
		},
		Handler: tool.handleDatabaseSecurity,
	}
}

func (tool *DatabaseTool) handleDatabaseQuery(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// First, check if database service is properly configured
	if err := tool.serviceManager.RequireService("database"); err != nil {
		tool.logger.Error("database service not configured", logging.String("error", err.Error()))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("❌ Database Error: %s\n\nTo fix this:\n1. Configure database settings in config.yaml\n2. Ensure all required fields are provided: host, port, username, password, dbname\n3. Verify database server is running and accessible", err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Check if database connection is healthy
	if !tool.db.IsConnected() {
		tool.logger.Error("database not connected")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "❌ Database Connection Error: Database is not connected\n\nThe system will attempt to reconnect automatically. Please check your database configuration and ensure the database server is running.",
				},
			},
			IsError: true,
		}, nil
	}

	// Perform health check
	if err := tool.db.HealthCheck(); err != nil {
		tool.logger.Error("database health check failed", logging.String("error", err.Error()))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("❌ Database Health Check Failed: %s\n\nThe database connection appears to be unstable. Please verify:\n1. Database server is running\n2. Network connectivity is stable\n3. Database credentials are correct", err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Extract query from request
	var args struct {
		Query string `json:"query"`
		Table string `json:"table,omitempty"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
	}

	if args.Query == "" {
		return createErrorResult(fmt.Errorf("query parameter is required")), nil
	}

	// Execute the query
	tool.logger.Info("executing database query", logging.String("query", args.Query))

	results, err := tool.db.Query(args.Query)
	if err != nil {
		tool.logger.Error("query execution failed", logging.String("error", err.Error()))

		// Check if it's a security validation error
		if strings.Contains(err.Error(), "SQL security validation failed") {
			// Try to get security info if it's an EnhancedDB
			if enhancedDB, ok := tool.db.(*database.EnhancedDB); ok {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: fmt.Sprintf("🚫 SQL Security Error: %s\n\n🔒 Security Policy:\n• Allowed operations: %s\n• Blocked operations: %s\n\n💡 Only read-only operations are permitted for security reasons.\nUse SELECT, SHOW, DESCRIBE, or EXPLAIN statements only.",
								err.Error(),
								strings.Join(enhancedDB.GetAllowedOperations(), ", "),
								strings.Join(enhancedDB.GetBlockedOperations(), ", ")),
						},
					},
					IsError: true,
				}, nil
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("❌ Query Execution Error: %s", err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Format results
	resultText := fmt.Sprintf("✅ Query executed successfully\n\nRows returned: %d\n\n", len(results))

	if len(results) == 0 {
		resultText += "No data returned."
	} else {
		// Show column headers
		if len(results) > 0 {
			var columns []string
			for col := range results[0] {
				columns = append(columns, col)
			}
			resultText += fmt.Sprintf("Columns: %v\n\n", columns)
		}

		// Show first 5 rows
		limit := len(results)
		if limit > 5 {
			limit = 5
		}

		resultText += "Sample data:\n"
		for i := 0; i < limit; i++ {
			resultText += fmt.Sprintf("Row %d: %v\n", i+1, results[i])
		}

		if len(results) > 5 {
			resultText += fmt.Sprintf("... and %d more rows\n", len(results)-5)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: resultText,
			},
		},
	}, nil
}

func (tool *DatabaseTool) handleDatabaseSecurity(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// First, check if database service is properly configured
	if err := tool.serviceManager.RequireService("database"); err != nil {
		tool.logger.Error("database service not configured", logging.String("error", err.Error()))
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("❌ Database Error: %s", err.Error()),
				},
			},
			IsError: true,
		}, nil
	}

	// Extract action from request
	var args struct {
		Action string `json:"action"`
	}

	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
	}

	if args.Action == "" {
		return createErrorResult(fmt.Errorf("action parameter is required")), nil
	}

	// Check if we have an EnhancedDB for security operations
	enhancedDB, ok := tool.db.(*database.EnhancedDB)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "❌ Security management is only available with enhanced database connections.",
				},
			},
			IsError: true,
		}, nil
	}

	// Execute the requested action
	tool.logger.Info("executing database security action", logging.String("action", args.Action))

	switch args.Action {
	case "status":
		return tool.getSecurityStatus(enhancedDB), nil
	case "enable_unsafe":
		return tool.enableUnsafeMode(enhancedDB), nil
	case "disable_unsafe":
		return tool.disableUnsafeMode(enhancedDB), nil
	case "allowed_ops":
		return tool.getAllowedOperations(enhancedDB), nil
	case "blocked_ops":
		return tool.getBlockedOperations(enhancedDB), nil
	default:
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("❌ Unknown action: %s. Available actions: status, enable_unsafe, disable_unsafe, allowed_ops, blocked_ops", args.Action),
				},
			},
			IsError: true,
		}, nil
	}
}

func (tool *DatabaseTool) getSecurityStatus(enhancedDB *database.EnhancedDB) *mcp.CallToolResult {
	unsafeMode := enhancedDB.IsUnsafeModeEnabled()
	allowedOps := enhancedDB.GetAllowedOperations()
	blockedOps := enhancedDB.GetBlockedOperations()

	statusIcon := "🔒"
	statusText := "SECURE"
	if unsafeMode {
		statusIcon = "⚠️"
		statusText = "UNSAFE"
	}

	text := fmt.Sprintf(`%s Database Security Status: %s

🔐 Current Security Configuration:
• Unsafe Mode: %t
• Allowed Operations: %s
• Blocked Operations: %s

%s Security Information:
%s

Available Actions:
• status - Show current security status
• enable_unsafe - Enable unsafe mode (allows all operations)
• disable_unsafe - Disable unsafe mode (secure defaults)
• allowed_ops - List allowed SQL operations
• blocked_ops - List blocked SQL operations`,
		statusIcon, statusText,
		unsafeMode,
		strings.Join(allowedOps, ", "),
		strings.Join(blockedOps, ", "),
		statusIcon,
		tool.getSecurityAdvice(unsafeMode))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
		IsError: false,
	}
}

func (tool *DatabaseTool) enableUnsafeMode(enhancedDB *database.EnhancedDB) *mcp.CallToolResult {
	tool.logger.Warn("enabling unsafe database mode")
	enhancedDB.EnableUnsafeMode()

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: `⚠️ UNSAFE MODE ENABLED

🚨 WARNING: All SQL operations are now allowed including:
• DELETE - Delete data
• DROP - Drop tables/databases
• UPDATE - Modify data
• TRUNCATE - Empty tables
• INSERT - Add data
• ALTER - Modify table structure
• CREATE - Create tables/databases

🔐 Security Recommendations:
1. Only use unsafe mode for administrative tasks
2. Disable unsafe mode immediately after use
3. Monitor all SQL operations carefully
4. Ensure proper backups are in place

Use 'disable_unsafe' action to return to secure mode.`,
			},
		},
		IsError: false,
	}
}

func (tool *DatabaseTool) disableUnsafeMode(enhancedDB *database.EnhancedDB) *mcp.CallToolResult {
	tool.logger.Info("disabling unsafe database mode")
	enhancedDB.DisableUnsafeMode()

	allowedOps := enhancedDB.GetAllowedOperations()
	blockedOps := enhancedDB.GetBlockedOperations()

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf(`🔒 SECURE MODE ENABLED

✅ Database security has been restored to safe defaults:

✅ Allowed Operations: %s
🚫 Blocked Operations: %s

🛡️ Security Features Active:
• Read-only operations only
• Dangerous operations blocked
• SQL injection protection
• Pattern-based security checks`,
					strings.Join(allowedOps, ", "),
					strings.Join(blockedOps, ", ")),
			},
		},
		IsError: false,
	}
}

func (tool *DatabaseTool) getAllowedOperations(enhancedDB *database.EnhancedDB) *mcp.CallToolResult {
	allowedOps := enhancedDB.GetAllowedOperations()

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf(`✅ Allowed SQL Operations:

• %s

These operations are permitted and considered safe for normal use.`,
					strings.Join(allowedOps, "\n• ")),
			},
		},
		IsError: false,
	}
}

func (tool *DatabaseTool) getBlockedOperations(enhancedDB *database.EnhancedDB) *mcp.CallToolResult {
	blockedOps := enhancedDB.GetBlockedOperations()

	if len(blockedOps) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: `⚠️ No operations are currently blocked.

This indicates that unsafe mode is enabled, which allows all SQL operations including potentially dangerous ones.

Consider using the 'disable_unsafe' action to enable security restrictions.`,
				},
			},
			IsError: false,
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf(`🚫 Blocked SQL Operations:

• %s

These operations are blocked for security reasons:
• They can modify or delete data
• They can alter database structure
• They pose security risks

Use 'enable_unsafe' action to temporarily allow these operations (not recommended for normal use).`,
					strings.Join(blockedOps, "\n• ")),
			},
		},
		IsError: false,
	}
}

func (tool *DatabaseTool) getSecurityAdvice(unsafeMode bool) string {
	if unsafeMode {
		return `🚨 UNSAFE MODE ACTIVE - All SQL operations are allowed
• Consider disabling unsafe mode for normal operations
• Monitor all queries carefully
• Ensure proper access controls are in place
• Have database backups ready`
	}

	return `🛡️ SECURE MODE ACTIVE - Only read-only operations allowed
• SELECT, SHOW, DESCRIBE, EXPLAIN are permitted
• Write operations are blocked for security
• This is the recommended setting for normal use
• Use unsafe mode only for administrative tasks`
}
