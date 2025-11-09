package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileSecurityValidator handles file operation security validation
type FileSecurityValidator struct {
	// Whitelisted directories - only operations within these directories are allowed
	whitelistedDirs []string
	// Whitelisted file extensions - only files with these extensions can be operated on
	whitelistedExtensions []string
	// Maximum file size allowed for operations (in bytes)
	maxFileSize int64
	// Read-only mode - if true, write operations are blocked
	readOnly bool
	// Mutex for thread safety
	mu sync.RWMutex
}

// NewFileSecurityValidator creates a new file security validator with default settings
func NewFileSecurityValidator(whitelistedDirs []string) *FileSecurityValidator {
	if whitelistedDirs == nil {
		// Default to current directory
		wd, _ := os.Getwd()
		whitelistedDirs = []string{wd}
	}

	return &FileSecurityValidator{
		whitelistedDirs:       whitelistedDirs,
		whitelistedExtensions: []string{"*"}, // Allow all extensions by default
		maxFileSize:           1024 * 1024,   // 1MB default limit
		readOnly:              false,         // Read-write mode by default
	}
}

// ValidateFileOperation validates if a file operation is allowed
func (v *FileSecurityValidator) ValidateFileOperation(operation, filePath string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Clean and resolve the file path
	cleanPath := filepath.Clean(filePath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if path is within whitelisted directories
	if !v.isPathWhitelisted(absPath) {
		return fmt.Errorf("file operation not allowed: path '%s' is outside whitelisted directories", filePath)
	}

	// Check for dangerous patterns
	if v.hasDangerousPatterns(cleanPath) {
		return fmt.Errorf("file operation not allowed: path contains dangerous patterns")
	}

	// Check file extension if not allowing all extensions
	if !v.isExtensionAllowed(cleanPath) {
		return fmt.Errorf("file operation not allowed: file extension not in whitelist")
	}

	// Check read-only mode for write operations
	if v.readOnly && (operation == "write" || operation == "create" || operation == "delete" || operation == "rename") {
		return fmt.Errorf("file operation not allowed: system is in read-only mode")
	}

	return nil
}

// ValidateFileSize validates if a file size is within allowed limits
func (v *FileSecurityValidator) ValidateFileSize(size int64) error {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if size > v.maxFileSize {
		return fmt.Errorf("file operation not allowed: file size (%d bytes) exceeds maximum allowed size (%d bytes)", size, v.maxFileSize)
	}
	return nil
}

// isPathWhitelisted checks if a path is within whitelisted directories
func (v *FileSecurityValidator) isPathWhitelisted(path string) bool {
	// Convert path to absolute if it's not already
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if path is within any whitelisted directory
	for _, dir := range v.whitelistedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}

		// Ensure the path is within or equal to the whitelisted directory
		rel, err := filepath.Rel(absDir, absPath)
		if err != nil {
			continue
		}

		// Check if the relative path goes up (../) - if it does, it's outside the directory
		if !strings.HasPrefix(rel, "..") && !strings.HasPrefix(rel, fmt.Sprintf("..%c", filepath.Separator)) {
			return true
		}
	}

	return false
}

// hasDangerousPatterns checks for dangerous path patterns
func (v *FileSecurityValidator) hasDangerousPatterns(path string) bool {
	// Check for absolute paths (already handled by filepath.Abs, but double-checking)
	if filepath.IsAbs(path) {
		return true
	}

	// Check for parent directory references
	if strings.Contains(path, "..") {
		return true
	}

	// Check for null bytes (can be used to bypass filters)
	if strings.Contains(path, "\x00") {
		return true
	}

	// Check for system directories (Windows)
	lowerPath := strings.ToLower(path)
	dangerousPaths := []string{
		"c:\\windows\\", "c:\\program files\\", "c:\\program files (x86)\\",
		"/etc/", "/bin/", "/sbin/", "/usr/bin/", "/usr/sbin/",
	}

	for _, dangerous := range dangerousPaths {
		if strings.Contains(lowerPath, dangerous) {
			return true
		}
	}

	return false
}

// isExtensionAllowed checks if a file extension is in the whitelist
func (v *FileSecurityValidator) isExtensionAllowed(path string) bool {
	// If whitelist contains "*", allow all extensions
	for _, ext := range v.whitelistedExtensions {
		if ext == "*" {
			return true
		}
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		ext = "no_extension"
	}

	// Check if extension is in whitelist
	for _, allowedExt := range v.whitelistedExtensions {
		if strings.ToLower(allowedExt) == ext {
			return true
		}
	}

	return false
}

// SetReadOnly sets the read-only mode
func (v *FileSecurityValidator) SetReadOnly(readOnly bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.readOnly = readOnly
}

// SetMaxFileSize sets the maximum allowed file size
func (v *FileSecurityValidator) SetMaxFileSize(size int64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.maxFileSize = size
}

// SetWhitelistedExtensions sets the whitelisted file extensions
func (v *FileSecurityValidator) SetWhitelistedExtensions(extensions []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.whitelistedExtensions = extensions
}

// AddWhitelistedDirectory adds a directory to the whitelist
func (v *FileSecurityValidator) AddWhitelistedDirectory(dir string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.whitelistedDirs = append(v.whitelistedDirs, dir)
}

// GetSecurityStatus returns the current security status
func (v *FileSecurityValidator) GetSecurityStatus() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return map[string]interface{}{
		"readonly":            v.readOnly,
		"max_file_size":       v.maxFileSize,
		"whitelisted_dirs":    v.whitelistedDirs,
		"whitelisted_exts":    v.whitelistedExtensions,
		"dangerous_patterns":  []string{"..", "\\x00", "system directories"},
	}

}