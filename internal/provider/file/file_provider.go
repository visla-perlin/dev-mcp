package file

import (
	"context"
	"dev-mcp/entity"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FileInfo represents file information
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Mode        string    `json:"mode"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
	Extension   string    `json:"extension,omitempty"`
	Permissions string    `json:"permissions"`
}

// FileProvider provides file system functionality
type FileProvider struct {
	allowedDirs []string
	readOnly    bool
	validator   *FileSecurityValidator
}

// NewFileProvider creates a new File provider with server
func NewFileProvider(server *mcp.Server) *FileProvider {
	// Create file security validator with default whitelisted directories
	validator := NewFileSecurityValidator([]string{"."})

	// Set read-only mode based on provider setting
	validator.SetReadOnly(true)

	p := &FileProvider{
		allowedDirs: []string{"."}, // 默认允许当前目录
		readOnly:    true,          // 默认只读模式
		validator:   validator,
	}

	// Add tools to server immediately
	p.addToolsToServer(server)
	log.Printf("✓ File provider initialized successfully")

	return p
}

// Test tests the file system access (for ProviderClient interface compatibility)
func (p *FileProvider) Test(config interface{}) error {
	// File provider is always available
	return nil
}

// AddTools adds File tools to the MCP server (for ProviderClient interface compatibility)
func (p *FileProvider) AddTools(server *mcp.Server) error {
	// Tools are already added in constructor, but we can call addToolsToServer again if needed
	p.addToolsToServer(server)
	return nil
}

// addToolsToServer adds File tools to the MCP server
func (p *FileProvider) addToolsToServer(server *mcp.Server) {
	// Add tools to server
	tools := []struct {
		tool    *mcp.Tool
		handler func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{p.createFileReadTool().Tool, p.createFileReadTool().Handler},
		{p.createFileWriteTool().Tool, p.createFileWriteTool().Handler},
		{p.createFileListTool().Tool, p.createFileListTool().Handler},
		{p.createFileDeleteTool().Tool, p.createFileDeleteTool().Handler},
		{p.createFileInfoTool().Tool, p.createFileInfoTool().Handler},
		{p.createFileRenameTool().Tool, p.createFileRenameTool().Handler},
	}

	for _, tool := range tools {
		server.AddTool(tool.tool, tool.handler)
		log.Printf("✓ Registered File tool: %s", tool.tool.Name)
	}

	log.Printf("✓ All File tools registered successfully")
}

// Close closes the File provider
func (p *FileProvider) Close() error {
	// File provider doesn't need explicit closing
	return nil
}

// validateWriteOperation validates if a write operation is allowed
func (p *FileProvider) validateWriteOperation() error {
	if p.readOnly {
		return fmt.Errorf("file system is in read-only mode")
	}
	return nil
}

// createFileReadTool creates the file read tool
func (p *FileProvider) createFileReadTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_read",
		Description: "Read file contents with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "File path to read"
				},
				"encoding": {
					"type": "string",
					"description": "File encoding (default: utf-8)",
					"default": "utf-8"
				}
			},
			"required": ["path"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path     string `json:"path"`
			Encoding string `json:"encoding,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Path == "" {
			return p.createErrorResult(fmt.Errorf("path parameter is required")), nil
		}

		if args.Encoding == "" {
			args.Encoding = "utf-8"
		}

		// Security validation using FileSecurityValidator
		if err := p.validator.ValidateFileOperation("read", args.Path); err != nil {
			return p.createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Check if file exists
		info, err := os.Stat(args.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return p.createErrorResult(fmt.Errorf("file does not exist: %s", args.Path)), nil
			}
			return p.createErrorResult(fmt.Errorf("failed to get file info: %w", err)), nil
		}

		// Validate file size using FileSecurityValidator
		if err := p.validator.ValidateFileSize(info.Size()); err != nil {
			return p.createErrorResult(fmt.Errorf("file size validation failed: %w", err)), nil
		}

		// Check if it's a directory
		if info.IsDir() {
			return p.createErrorResult(fmt.Errorf("path is a directory, not a file: %s", args.Path)), nil
		}

		// Read file
		content, err := os.ReadFile(args.Path)
		if err != nil {
			return p.createErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
		}

		// Limit file size for security (1MB max)
		if len(content) > 1024*1024 {
			return p.createErrorResult(fmt.Errorf("file too large (max 1MB)")), nil
		}

		result := map[string]interface{}{
			"path":     args.Path,
			"content":  string(content),
			"size":     len(content),
			"encoding": args.Encoding,
			"mod_time": info.ModTime(),
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createFileWriteTool creates the file write tool
func (p *FileProvider) createFileWriteTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_write",
		Description: "Write content to a file with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to the file to write"
				},
				"content": {
					"type": "string",
					"description": "Content to write to the file"
				},
				"append": {
					"type": "boolean",
					"description": "Whether to append to existing file (default: false)",
					"default": false
				},
				"create_dirs": {
					"type": "boolean",
					"description": "Whether to create parent directories if they don't exist (default: false)",
					"default": false
				}
			},
			"required": ["path", "content"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path       string `json:"path"`
			Content    string `json:"content"`
			Append     bool   `json:"append,omitempty"`
			CreateDirs bool   `json:"create_dirs,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Security validation using FileSecurityValidator
		if err := p.validator.ValidateFileOperation("write", args.Path); err != nil {
			return p.createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Validate write operation
		if err := p.validateWriteOperation(); err != nil {
			return p.createErrorResult(fmt.Errorf("write operation not allowed: %w", err)), nil
		}

		// Validate file size using FileSecurityValidator
		if err := p.validator.ValidateFileSize(int64(len(args.Content))); err != nil {
			return p.createErrorResult(fmt.Errorf("file size validation failed: %w", err)), nil
		}

		// Create parent directories if requested
		if args.CreateDirs {
			dir := filepath.Dir(args.Path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return p.createErrorResult(fmt.Errorf("failed to create directories: %w", err)), nil
			}
		}

		// Write file
		var err error
		if args.Append {
			file, err := os.OpenFile(args.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return p.createErrorResult(fmt.Errorf("failed to open file for append: %w", err)), nil
			}
			defer file.Close()

			if _, err = file.WriteString(args.Content); err != nil {
				return p.createErrorResult(fmt.Errorf("failed to write to file: %w", err)), nil
			}
		} else {
			err = os.WriteFile(args.Path, []byte(args.Content), 0644)
		}

		if err != nil {
			return p.createErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
		}

		// Get file info after write
		info, err := os.Stat(args.Path)
		if err != nil {
			return p.createErrorResult(fmt.Errorf("failed to get file info: %w", err)), nil
		}

		result := map[string]interface{}{
			"path":          args.Path,
			"size":          info.Size(),
			"written_bytes": len(args.Content),
			"append":        args.Append,
			"mod_time":      info.ModTime(),
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createFileListTool creates the file list tool
func (p *FileProvider) createFileListTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_list",
		Description: "List files in directory with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Directory path to list",
					"default": "."
				},
				"pattern": {
					"type": "string",
					"description": "File pattern to match (optional)"
				},
				"recursive": {
					"type": "boolean",
					"description": "Whether to list recursively (default: false)",
					"default": false
				}
			}
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path      string `json:"path,omitempty"`
			Pattern   string `json:"pattern,omitempty"`
			Recursive bool   `json:"recursive,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Path == "" {
			args.Path = "."
		}

		// Security validation using FileSecurityValidator
		if err := p.validator.ValidateFileOperation("read", args.Path); err != nil {
			return p.createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Check if directory exists
		info, err := os.Stat(args.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return p.createErrorResult(fmt.Errorf("directory does not exist: %s", args.Path)), nil
			}
			return p.createErrorResult(fmt.Errorf("failed to get directory info: %w", err)), nil
		}

		// Check if it's a directory
		if !info.IsDir() {
			return p.createErrorResult(fmt.Errorf("path is not a directory: %s", args.Path)), nil
		}

		var files []FileInfo

		if args.Recursive {
			// Recursive listing
			err = filepath.Walk(args.Path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					// Skip files that cause errors
					return nil
				}

				// Skip the root directory itself
				if filePath == args.Path {
					return nil
				}

				// Apply pattern filter if specified
				if args.Pattern != "" {
					matched, err := filepath.Match(args.Pattern, info.Name())
					if err != nil || !matched {
						return nil
					}
				}

				// Security validation for each file using FileSecurityValidator
				if err := p.validator.ValidateFileOperation("read", filePath); err != nil {
					// Skip files that fail validation
					return nil
				}

				fileInfo := FileInfo{
					Name:        info.Name(),
					Path:        filePath,
					Size:        info.Size(),
					Mode:        info.Mode().String(),
					ModTime:     info.ModTime(),
					IsDir:       info.IsDir(),
					Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
				}

				if !info.IsDir() {
					fileInfo.Extension = filepath.Ext(info.Name())
				}

				files = append(files, fileInfo)
				return nil
			})
		} else {
			// Non-recursive listing
			entries, err := os.ReadDir(args.Path)
			if err != nil {
				return p.createErrorResult(fmt.Errorf("failed to read directory: %w", err)), nil
			}

			for _, entry := range entries {
				// Apply pattern filter if specified
				if args.Pattern != "" {
					matched, err := filepath.Match(args.Pattern, entry.Name())
					if err != nil || !matched {
						continue
					}
				}

				fullPath := filepath.Join(args.Path, entry.Name())
				info, err := entry.Info()
				if err != nil {
					continue
				}

				// Security validation using FileSecurityValidator
				if err := p.validator.ValidateFileOperation("read", fullPath); err != nil {
					// Skip files that fail validation
					continue
				}

				fileInfo := FileInfo{
					Name:        info.Name(),
					Path:        fullPath,
					Size:        info.Size(),
					Mode:        info.Mode().String(),
					ModTime:     info.ModTime(),
					IsDir:       info.IsDir(),
					Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
				}

				if !info.IsDir() {
					fileInfo.Extension = filepath.Ext(info.Name())
				}

				files = append(files, fileInfo)
			}
		}

		if err != nil {
			return p.createErrorResult(fmt.Errorf("failed to list directory: %w", err)), nil
		}

		result := map[string]interface{}{
			"path":      args.Path,
			"files":     files,
			"count":     len(files),
			"recursive": args.Recursive,
			"pattern":   args.Pattern,
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createFileDeleteTool creates the file delete tool
func (p *FileProvider) createFileDeleteTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_delete",
		Description: "Delete a file or directory with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to the file or directory to delete"
				},
				"recursive": {
					"type": "boolean",
					"description": "Whether to delete directories recursively (default: false)",
					"default": false
				}
			},
			"required": ["path"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path      string `json:"path"`
			Recursive bool   `json:"recursive,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Security validation using FileSecurityValidator
		if err := p.validator.ValidateFileOperation("delete", args.Path); err != nil {
			return p.createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Validate write operation
		if err := p.validateWriteOperation(); err != nil {
			return p.createErrorResult(fmt.Errorf("delete operation not allowed: %w", err)), nil
		}

		// Check if file exists
		info, err := os.Stat(args.Path)
		if os.IsNotExist(err) {
			return p.createErrorResult(fmt.Errorf("file does not exist: %s", args.Path)), nil
		}

		// Check if it's a directory and recursive is needed
		if info.IsDir() && !args.Recursive {
			return p.createErrorResult(fmt.Errorf("path is a directory, use recursive=true to delete directories")), nil
		}

		// Delete file or directory
		var deleteErr error
		if args.Recursive {
			deleteErr = os.RemoveAll(args.Path)
		} else {
			deleteErr = os.Remove(args.Path)
		}

		if deleteErr != nil {
			return p.createErrorResult(fmt.Errorf("failed to delete: %w", deleteErr)), nil
		}

		result := map[string]interface{}{
			"path":      args.Path,
			"deleted":   true,
			"was_dir":   info.IsDir(),
			"recursive": args.Recursive,
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createFileInfoTool creates the file info tool
func (p *FileProvider) createFileInfoTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_info",
		Description: "Get information about a file or directory",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to the file or directory"
				}
			},
			"required": ["path"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path string `json:"path"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Security validation using FileSecurityValidator
		if err := p.validator.ValidateFileOperation("read", args.Path); err != nil {
			return p.createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Get file info
		info, err := os.Stat(args.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return p.createErrorResult(fmt.Errorf("file does not exist: %s", args.Path)), nil
			}
			return p.createErrorResult(fmt.Errorf("failed to get file info: %w", err)), nil
		}

		absPath, _ := filepath.Abs(args.Path)

		fileInfo := FileInfo{
			Name:        info.Name(),
			Path:        absPath,
			Size:        info.Size(),
			Mode:        info.Mode().String(),
			ModTime:     info.ModTime(),
			IsDir:       info.IsDir(),
			Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
		}

		if !info.IsDir() {
			fileInfo.Extension = filepath.Ext(info.Name())
		}

		return p.formatJSONResult(fileInfo), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// createFileRenameTool creates the file rename/move tool
func (p *FileProvider) createFileRenameTool() entity.ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_rename",
		Description: "Rename or move a file/directory with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"old_path": {
					"type": "string",
					"description": "Current path of the file or directory"
				},
				"new_path": {
					"type": "string",
					"description": "New path for the file or directory"
				}
			},
			"required": ["old_path", "new_path"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return p.createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Security validation for both paths using FileSecurityValidator
		if err := p.validator.ValidateFileOperation("read", args.OldPath); err != nil {
			return p.createErrorResult(fmt.Errorf("source path security validation failed: %w", err)), nil
		}

		if err := p.validator.ValidateFileOperation("write", args.NewPath); err != nil {
			return p.createErrorResult(fmt.Errorf("destination path security validation failed: %w", err)), nil
		}

		// Validate write operation
		if err := p.validateWriteOperation(); err != nil {
			return p.createErrorResult(fmt.Errorf("rename operation not allowed: %w", err)), nil
		}

		// Check if source exists
		info, err := os.Stat(args.OldPath)
		if os.IsNotExist(err) {
			return p.createErrorResult(fmt.Errorf("source file does not exist: %s", args.OldPath)), nil
		}

		// Check if destination already exists
		if _, err := os.Stat(args.NewPath); err == nil {
			return p.createErrorResult(fmt.Errorf("destination already exists: %s", args.NewPath)), nil
		}

		// Rename/move file
		if err := os.Rename(args.OldPath, args.NewPath); err != nil {
			return p.createErrorResult(fmt.Errorf("failed to rename/move: %w", err)), nil
		}

		result := map[string]interface{}{
			"old_path": args.OldPath,
			"new_path": args.NewPath,
			"is_dir":   info.IsDir(),
			"size":     info.Size(),
		}

		return p.formatJSONResult(result), nil
	}

	return entity.ToolDefinition{Tool: tool, Handler: handler}
}

// Helper functions
func (p *FileProvider) createErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("File Error: %v", err)}},
		IsError: true,
	}
}

func (p *FileProvider) formatJSONResult(data interface{}) *mcp.CallToolResult {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return p.createErrorResult(fmt.Errorf("failed to marshal data: %w", err))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}
}
