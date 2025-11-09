package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/entity"
	"dev-mcp/internal/config"
	"dev-mcp/internal/provider"
)

// DatabaseProvider provides database query functionality
type DatabaseProvider struct {
	*provider.BaseProvider
	client *DatabaseClient
}

// NewDatabaseProvider creates a new Database provider with config
func NewDatabaseProvider(cfg *config.DatabaseConfig) *DatabaseProvider {
	p := &DatabaseProvider{
		BaseProvider: provider.NewBaseProvider("database"),
	}

	// Try to create database client
	client, err := NewDatabaseClient(cfg)
	if err != nil {
		log.Printf("⚠ Database client initialization failed: %v", err)
		p.SetStatus(false, "Database client initialization failed", err)
		return p
	}

	p.client = client
	p.SetAvailable(true)
	log.Printf("✓ Database provider initialized successfully")
	return p
}

// Test tests the database configuration and connection (for ProviderClient interface compatibility)
func (p *DatabaseProvider) Test(config interface{}) error {
	// Since client is already initialized in constructor, just check availability
	if !p.IsAvailable() {
		return fmt.Errorf("database provider not available")
	}
	return nil
}

// AddTools adds database tools to the MCP server (for ProviderClient interface compatibility)
func (p *DatabaseProvider) AddTools(server *mcp.Server, config interface{}) error {
	// Register tools with the server directly
	toolDef1 := p.createDatabaseQueryTool()
	server.AddTool(toolDef1.Tool, toolDef1.Handler)

	toolDef2 := p.createDatabaseSecurityTool()
	server.AddTool(toolDef2.Tool, toolDef2.Handler)

	log.Printf("✓ Database tools added to server successfully")
	return nil
}

// Close closes the Database provider
func (p *DatabaseProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// HealthCheck performs health check for Database
func (p *DatabaseProvider) HealthCheck() error {
	if !p.IsAvailable() {
		return fmt.Errorf("database provider not available")
	}

	if p.client == nil {
		return fmt.Errorf("database client not initialized")
	}

	if err := p.client.HealthCheck(); err != nil {
		p.SetStatus(false, "Database health check failed", err)
		return err
	}

	p.SetStatus(true, "Database healthy", nil)
	return nil
}

// createDatabaseQueryTool creates the database query tool
func (p *DatabaseProvider) createDatabaseQueryTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "database_query",
		Description: "Execute secure database queries and manage database operations. Only read-only operations are allowed by default (SELECT, SHOW, DESCRIBE, EXPLAIN). Write operations are blocked for security unless unsafe mode is enabled.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "SQL query to execute (read-only operations only by default)"
				}
			},
			"required": ["query"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract query from request
		var args struct {
			Query string `json:"query"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Query == "" {
			return p.createErrorResult(fmt.Errorf("query parameter is required")), nil
		}

		// Execute the query
		log.Printf("Executing database query: %s", args.Query)
		results, err := p.client.Query(args.Query)
		if err != nil {
			log.Printf("Query execution failed: %v", err)

			// Check if it's a security validation error
			if strings.Contains(err.Error(), "SQL security validation failed") {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{
							Text: fmt.Sprintf("🚫 SQL Security Error: %s\n\n🔒 Security Policy:\n• Allowed operations: %s\n• Blocked operations: %s\n\n💡 Only read-only operations are permitted for security reasons.\nUse SELECT, SHOW, DESCRIBE, or EXPLAIN statements only.",
								err.Error(),
								strings.Join(p.client.GetAllowedOperations(), ", "),
								strings.Join(p.client.GetBlockedOperations(), ", ")),
						},
					},
					IsError: true,
				}, nil
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

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createDatabaseSecurityTool creates the database security management tool
func (p *DatabaseProvider) createDatabaseSecurityTool() entity.ToolDefinition {
	tool := &mcp.Tool{
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
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract action from request
		var args struct {
			Action string `json:"action"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Action == "" {
			return p.createErrorResult(fmt.Errorf("action parameter is required")), nil
		}

		// Execute the requested action
		log.Printf("Executing database security action: %s", args.Action)

		switch args.Action {
		case "status":
			return p.getSecurityStatus(), nil
		case "enable_unsafe":
			return p.enableUnsafeMode(), nil
		case "disable_unsafe":
			return p.disableUnsafeMode(), nil
		case "allowed_ops":
			return p.getAllowedOperations(), nil
		case "blocked_ops":
			return p.getBlockedOperations(), nil
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

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

func (p *DatabaseProvider) getSecurityStatus() *mcp.CallToolResult {
	unsafeMode := p.client.IsUnsafeModeEnabled()
	allowedOps := p.client.GetAllowedOperations()
	blockedOps := p.client.GetBlockedOperations()

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
		p.getSecurityAdvice(unsafeMode))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
		IsError: false,
	}
}

func (p *DatabaseProvider) enableUnsafeMode() *mcp.CallToolResult {
	log.Printf("Enabling unsafe database mode")
	p.client.EnableUnsafeMode()

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

func (p *DatabaseProvider) disableUnsafeMode() *mcp.CallToolResult {
	log.Printf("Disabling unsafe database mode")
	p.client.DisableUnsafeMode()

	allowedOps := p.client.GetAllowedOperations()
	blockedOps := p.client.GetBlockedOperations()

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

func (p *DatabaseProvider) getAllowedOperations() *mcp.CallToolResult {
	allowedOps := p.client.GetAllowedOperations()

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

func (p *DatabaseProvider) getBlockedOperations() *mcp.CallToolResult {
	blockedOps := p.client.GetBlockedOperations()

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

func (p *DatabaseProvider) getSecurityAdvice(unsafeMode bool) string {
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

// Helper functions
func (p *DatabaseProvider) createErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Database Error: %v", err)}},
		IsError: true,
	}
}

func (p *DatabaseProvider) formatJSONResult(data interface{}) *mcp.CallToolResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return p.createErrorResult(fmt.Errorf("failed to marshal data: %w", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}
}
