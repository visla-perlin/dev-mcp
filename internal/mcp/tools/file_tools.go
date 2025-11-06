package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"dev-mcp/internal/file"
)

// FileManager handles secure file operations
type FileManager struct {
	validator *file.FileSecurityValidator
}

// NewFileManager creates a new file manager with security validation
func NewFileManager() *FileManager {
	return &FileManager{
		validator: file.NewFileSecurityValidator(nil),
	}
}

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

// FileReadTool creates a tool for reading files
func NewFileReadTool(fm *FileManager) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_read",
		Description: "Read contents of a file with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to the file to read"
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
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		if args.Encoding == "" {
			args.Encoding = "utf-8"
		}

		// Validate file operation
		if err := fm.validator.ValidateFileOperation("read", args.Path); err != nil {
			return createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Check if file exists
		if _, err := os.Stat(args.Path); os.IsNotExist(err) {
			return createErrorResult(fmt.Errorf("file does not exist: %s", args.Path)), nil
		}

		// Read file
		content, err := os.ReadFile(args.Path)
		if err != nil {
			return createErrorResult(fmt.Errorf("failed to read file: %w", err)), nil
		}

		// Validate file size
		if err := fm.validator.ValidateFileSize(int64(len(content))); err != nil {
			return createErrorResult(err), nil
		}

		result := map[string]interface{}{
			"path":     args.Path,
			"content":  string(content),
			"size":     len(content),
			"encoding": args.Encoding,
		}

		return formatJSONResult(result), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// FileWriteTool creates a tool for writing files
func NewFileWriteTool(fm *FileManager) ToolDefinition {
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
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Validate file operation
		operation := "write"
		if _, err := os.Stat(args.Path); os.IsNotExist(err) {
			operation = "create"
		}

		if err := fm.validator.ValidateFileOperation(operation, args.Path); err != nil {
			return createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Validate content size
		if err := fm.validator.ValidateFileSize(int64(len(args.Content))); err != nil {
			return createErrorResult(err), nil
		}

		// Create parent directories if requested
		if args.CreateDirs {
			dir := filepath.Dir(args.Path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return createErrorResult(fmt.Errorf("failed to create directories: %w", err)), nil
			}
		}

		// Write file
		var err error
		if args.Append {
			file, err := os.OpenFile(args.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return createErrorResult(fmt.Errorf("failed to open file for append: %w", err)), nil
			}
			defer file.Close()

			if _, err = file.WriteString(args.Content); err != nil {
				return createErrorResult(fmt.Errorf("failed to write to file: %w", err)), nil
			}
		} else {
			err = os.WriteFile(args.Path, []byte(args.Content), 0644)
		}

		if err != nil {
			return createErrorResult(fmt.Errorf("failed to write file: %w", err)), nil
		}

		// Get file info after write
		info, err := os.Stat(args.Path)
		if err != nil {
			return createErrorResult(fmt.Errorf("failed to get file info: %w", err)), nil
		}

		result := map[string]interface{}{
			"path":          args.Path,
			"size":          info.Size(),
			"written_bytes": len(args.Content),
			"append":        args.Append,
			"mod_time":      info.ModTime(),
		}

		return formatJSONResult(result), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// FileInfoTool creates a tool for getting file information
func NewFileInfoTool(fm *FileManager) ToolDefinition {
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
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Validate file operation
		if err := fm.validator.ValidateFileOperation("read", args.Path); err != nil {
			return createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Get file info
		info, err := os.Stat(args.Path)
		if err != nil {
			return createErrorResult(fmt.Errorf("failed to get file info: %w", err)), nil
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

		return formatJSONResult(fileInfo), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// FileDeleteTool creates a tool for deleting files
func NewFileDeleteTool(fm *FileManager) ToolDefinition {
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
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Validate file operation
		if err := fm.validator.ValidateFileOperation("delete", args.Path); err != nil {
			return createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Check if file exists
		info, err := os.Stat(args.Path)
		if os.IsNotExist(err) {
			return createErrorResult(fmt.Errorf("file does not exist: %s", args.Path)), nil
		}

		// Check if it's a directory and recursive is needed
		if info.IsDir() && !args.Recursive {
			return createErrorResult(fmt.Errorf("path is a directory, use recursive=true to delete directories")), nil
		}

		// Delete file or directory
		var deleteErr error
		if args.Recursive {
			deleteErr = os.RemoveAll(args.Path)
		} else {
			deleteErr = os.Remove(args.Path)
		}

		if deleteErr != nil {
			return createErrorResult(fmt.Errorf("failed to delete: %w", deleteErr)), nil
		}

		result := map[string]interface{}{
			"path":      args.Path,
			"deleted":   true,
			"was_dir":   info.IsDir(),
			"recursive": args.Recursive,
		}

		return formatJSONResult(result), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// FileRenameTool creates a tool for renaming/moving files
func NewFileRenameTool(fm *FileManager) ToolDefinition {
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
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Validate both paths
		if err := fm.validator.ValidateFileOperation("rename", args.OldPath); err != nil {
			return createErrorResult(fmt.Errorf("source path security validation failed: %w", err)), nil
		}

		if err := fm.validator.ValidateFileOperation("create", args.NewPath); err != nil {
			return createErrorResult(fmt.Errorf("destination path security validation failed: %w", err)), nil
		}

		// Check if source exists
		info, err := os.Stat(args.OldPath)
		if os.IsNotExist(err) {
			return createErrorResult(fmt.Errorf("source file does not exist: %s", args.OldPath)), nil
		}

		// Check if destination already exists
		if _, err := os.Stat(args.NewPath); err == nil {
			return createErrorResult(fmt.Errorf("destination already exists: %s", args.NewPath)), nil
		}

		// Rename/move file
		if err := os.Rename(args.OldPath, args.NewPath); err != nil {
			return createErrorResult(fmt.Errorf("failed to rename/move: %w", err)), nil
		}

		result := map[string]interface{}{
			"old_path": args.OldPath,
			"new_path": args.NewPath,
			"is_dir":   info.IsDir(),
			"size":     info.Size(),
		}

		return formatJSONResult(result), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// FileListTool creates a tool for listing directory contents
func NewFileListTool(fm *FileManager) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_list",
		Description: "List contents of a directory with security validation",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Path to the directory to list"
				},
				"recursive": {
					"type": "boolean",
					"description": "Whether to list recursively (default: false)",
					"default": false
				},
				"pattern": {
					"type": "string",
					"description": "File pattern to match (e.g., '*.txt')"
				}
			},
			"required": ["path"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Path      string `json:"path"`
			Recursive bool   `json:"recursive,omitempty"`
			Pattern   string `json:"pattern,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		// Validate file operation
		if err := fm.validator.ValidateFileOperation("read", args.Path); err != nil {
			return createErrorResult(fmt.Errorf("security validation failed: %w", err)), nil
		}

		// Check if directory exists
		info, err := os.Stat(args.Path)
		if os.IsNotExist(err) {
			return createErrorResult(fmt.Errorf("directory does not exist: %s", args.Path)), nil
		}

		if !info.IsDir() {
			return createErrorResult(fmt.Errorf("path is not a directory: %s", args.Path)), nil
		}

		var files []FileInfo

		if args.Recursive {
			err = filepath.Walk(args.Path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip validation for subdirectories as we already validated the root
				if filePath != args.Path {
					if err := fm.validator.ValidateFileOperation("read", filePath); err != nil {
						// Skip files that fail validation but don't stop the whole operation
						return nil
					}
				}

				// Apply pattern filter if specified
				if args.Pattern != "" {
					matched, err := filepath.Match(args.Pattern, info.Name())
					if err != nil || !matched {
						return nil
					}
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
			entries, err := os.ReadDir(args.Path)
			if err != nil {
				return createErrorResult(fmt.Errorf("failed to read directory: %w", err)), nil
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
			return createErrorResult(fmt.Errorf("failed to list directory: %w", err)), nil
		}

		result := map[string]interface{}{
			"path":      args.Path,
			"files":     files,
			"count":     len(files),
			"recursive": args.Recursive,
			"pattern":   args.Pattern,
		}

		return formatJSONResult(result), nil
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// FileSecurityTool creates a tool for managing file security settings
func NewFileSecurityTool(fm *FileManager) ToolDefinition {
	tool := &mcp.Tool{
		Name:        "file_security",
		Description: "Manage file security settings and policies",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"description": "Action to perform: status, set_readonly, add_extension, remove_extension",
					"enum": ["status", "set_readonly", "add_extension", "remove_extension"]
				},
				"readonly": {
					"type": "boolean",
					"description": "Set read-only mode (for set_readonly action)"
				},
				"extension": {
					"type": "string",
					"description": "File extension to add/remove (for add_extension/remove_extension actions)"
				}
			},
			"required": ["action"]
		}`),
	}

	handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			Action    string `json:"action"`
			ReadOnly  bool   `json:"readonly,omitempty"`
			Extension string `json:"extension,omitempty"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return createErrorResult(fmt.Errorf("invalid arguments: %w", err)), nil
		}

		switch args.Action {
		case "status":
			status := fm.validator.GetSecurityStatus()
			return formatJSONResult(status), nil

		case "set_readonly":
			// readonly is a bool, so it will be false if not present, which is a safe default
			fm.validator.SetReadOnly(args.ReadOnly)

			result := map[string]interface{}{
				"action":   "set_readonly",
				"readonly": args.ReadOnly,
				"status":   "success",
			}
			return formatJSONResult(result), nil

		case "add_extension":
			if args.Extension == "" {
				return createErrorResult(fmt.Errorf("extension parameter is required for add_extension action")), nil
			}
			fm.validator.AddAllowedExtension(args.Extension)

			result := map[string]interface{}{
				"action":    "add_extension",
				"extension": args.Extension,
				"status":    "success",
			}
			return formatJSONResult(result), nil

		case "remove_extension":
			if args.Extension == "" {
				return createErrorResult(fmt.Errorf("extension parameter is required for remove_extension action")), nil
			}
			fm.validator.RemoveAllowedExtension(args.Extension)

			result := map[string]interface{}{
				"action":    "remove_extension",
				"extension": args.Extension,
				"status":    "success",
			}
			return formatJSONResult(result), nil

		default:
			return createErrorResult(fmt.Errorf("unknown action: %s", args.Action)), nil
		}
	}

	return ToolDefinition{Tool: tool, Handler: handler}
}

// RegisterFileTools registers all file-related tools
func RegisterFileTools(registrar ToolRegistrar) {
	fileManager := NewFileManager()

	registrar.RegisterTool(NewFileReadTool(fileManager))
	registrar.RegisterTool(NewFileWriteTool(fileManager))
	registrar.RegisterTool(NewFileInfoTool(fileManager))
	registrar.RegisterTool(NewFileDeleteTool(fileManager))
	registrar.RegisterTool(NewFileRenameTool(fileManager))
	registrar.RegisterTool(NewFileListTool(fileManager))
	registrar.RegisterTool(NewFileSecurityTool(fileManager))
}
