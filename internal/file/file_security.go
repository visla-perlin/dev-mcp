package file

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// FileSecurityPolicy defines security policies for file operations
type FileSecurityPolicy struct {
	AllowedExtensions []string
	BlockedPaths      []string
	MaxFileSize       int64 // in bytes
	ReadOnly          bool
}

// DefaultFileSecurityPolicy returns a default security policy
func DefaultFileSecurityPolicy() *FileSecurityPolicy {
	return &FileSecurityPolicy{
		AllowedExtensions: []string{".txt", ".md", ".json", ".yaml", ".yml", ".log", ".csv", ".xml"},
		BlockedPaths:      GetDangerousPaths(),
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		ReadOnly:          false,
	}
}

// GetDangerousPaths returns a list of dangerous system paths for all platforms
func GetDangerousPaths() []string {
	commonDangerous := []string{
		// Common dangerous patterns
		"passwd", "shadow", "hosts", "sudoers",
		".ssh", ".aws", ".config",
		"id_rsa", "id_dsa", "id_ed25519",
		"private", "secret", "key",
	}

	switch runtime.GOOS {
	case "windows":
		return append(commonDangerous, []string{
			// Windows system directories
			"C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)",
			"C:\\System Volume Information", "C:\\ProgramData\\Microsoft",
			"C:\\Users\\Default", "C:\\Users\\Public",
			// Windows registry and system files
			"\\System32", "\\SysWOW64", "\\boot", "\\Recovery",
			"\\$Recycle.Bin", "\\pagefile.sys", "\\swapfile.sys",
			"\\hiberfil.sys", "\\bootmgr",
			// Windows config and security
			"\\SAM", "\\SECURITY", "\\SOFTWARE", "\\SYSTEM",
			"AppData\\Roaming\\Microsoft", "AppData\\Local\\Microsoft",
		}...)
	case "darwin":
		return append(commonDangerous, []string{
			// macOS system directories
			"/System", "/Library/System", "/usr/bin", "/usr/sbin",
			"/bin", "/sbin", "/etc", "/var/log", "/var/db",
			"/Library/Application Support", "/Library/Preferences",
			"/Library/Keychains", "/Library/Security",
			// macOS specific
			"/Applications/Utilities", "/System/Library",
			"/Library/StartupItems", "/Library/LaunchDaemons",
			"/Library/LaunchAgents", "/private/var",
		}...)
	case "linux":
		return append(commonDangerous, []string{
			// Linux system directories
			"/etc", "/usr/bin", "/usr/sbin", "/bin", "/sbin",
			"/boot", "/proc", "/sys", "/dev", "/run",
			"/var/log", "/var/lib", "/var/cache", "/var/spool",
			"/root", "/lib", "/lib64", "/opt/system",
			// Linux security and config
			"/etc/passwd", "/etc/shadow", "/etc/hosts",
			"/etc/sudoers", "/etc/ssh", "/etc/ssl",
			"/etc/security", "/etc/systemd",
		}...)
	default:
		return commonDangerous
	}
}

// FileSecurityValidator validates file operations for security
type FileSecurityValidator struct {
	Policy *FileSecurityPolicy
}

// NewFileSecurityValidator creates a new file security validator
func NewFileSecurityValidator(policy *FileSecurityPolicy) *FileSecurityValidator {
	if policy == nil {
		policy = DefaultFileSecurityPolicy()
	}
	return &FileSecurityValidator{Policy: policy}
}

// ValidateFileOperation validates if a file operation is safe
func (v *FileSecurityValidator) ValidateFileOperation(operation string, filePath string) error {
	// Normalize path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Check for dangerous paths
	if err := v.checkDangerousPath(absPath); err != nil {
		return err
	}

	// Check file extension for write operations
	if operation == "write" || operation == "create" {
		if err := v.checkAllowedExtension(absPath); err != nil {
			return err
		}
	}

	// Check read-only policy for write operations
	if (operation == "write" || operation == "create" || operation == "delete" || operation == "rename") && v.Policy.ReadOnly {
		return fmt.Errorf("file system is in read-only mode, %s operation not allowed", operation)
	}

	return nil
}

// ValidateFileSize validates file size against policy
func (v *FileSecurityValidator) ValidateFileSize(size int64) error {
	if size > v.Policy.MaxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size %d bytes", size, v.Policy.MaxFileSize)
	}
	return nil
}

// checkDangerousPath checks if the path contains dangerous patterns
func (v *FileSecurityValidator) checkDangerousPath(absPath string) error {
	normalizedPath := strings.ToLower(filepath.ToSlash(absPath))

	for _, dangerousPath := range v.Policy.BlockedPaths {
		dangerousPattern := strings.ToLower(dangerousPath)

		// Check if path contains dangerous pattern
		if strings.Contains(normalizedPath, dangerousPattern) {
			return fmt.Errorf("access denied: path '%s' contains blocked pattern '%s'", absPath, dangerousPath)
		}

		// Check if path starts with dangerous directory
		if strings.HasPrefix(normalizedPath, dangerousPattern) {
			return fmt.Errorf("access denied: path '%s' is in blocked directory '%s'", absPath, dangerousPath)
		}
	}

	return nil
}

// checkAllowedExtension checks if file extension is allowed
func (v *FileSecurityValidator) checkAllowedExtension(absPath string) error {
	ext := strings.ToLower(filepath.Ext(absPath))

	// Allow files without extension for now
	if ext == "" {
		return nil
	}

	for _, allowedExt := range v.Policy.AllowedExtensions {
		if ext == strings.ToLower(allowedExt) {
			return nil
		}
	}

	return fmt.Errorf("file extension '%s' is not allowed. Allowed extensions: %v", ext, v.Policy.AllowedExtensions)
}

// GetSecurityStatus returns current security policy status
func (v *FileSecurityValidator) GetSecurityStatus() map[string]interface{} {
	return map[string]interface{}{
		"read_only":           v.Policy.ReadOnly,
		"max_file_size":       v.Policy.MaxFileSize,
		"allowed_extensions":  v.Policy.AllowedExtensions,
		"blocked_paths_count": len(v.Policy.BlockedPaths),
		"platform":            runtime.GOOS,
	}
}

// SetReadOnly sets the read-only mode
func (v *FileSecurityValidator) SetReadOnly(readOnly bool) {
	v.Policy.ReadOnly = readOnly
}

// AddAllowedExtension adds an allowed file extension
func (v *FileSecurityValidator) AddAllowedExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// Check if already exists
	for _, existing := range v.Policy.AllowedExtensions {
		if strings.EqualFold(existing, ext) {
			return
		}
	}

	v.Policy.AllowedExtensions = append(v.Policy.AllowedExtensions, strings.ToLower(ext))
}

// RemoveAllowedExtension removes an allowed file extension
func (v *FileSecurityValidator) RemoveAllowedExtension(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	for i, existing := range v.Policy.AllowedExtensions {
		if strings.EqualFold(existing, ext) {
			v.Policy.AllowedExtensions = append(v.Policy.AllowedExtensions[:i], v.Policy.AllowedExtensions[i+1:]...)
			return
		}
	}
}
