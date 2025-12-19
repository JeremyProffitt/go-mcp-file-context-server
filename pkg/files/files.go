package files

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// FileMetadata represents metadata about a file
type FileMetadata struct {
	Size         int64     `json:"size"`
	MimeType     string    `json:"mimeType"`
	ModifiedTime time.Time `json:"modifiedTime"`
	CreatedTime  time.Time `json:"createdTime"`
	IsDirectory  bool      `json:"isDirectory"`
}

// FileContent represents file content with metadata
type FileContent struct {
	Content    string       `json:"content"`
	Metadata   FileMetadata `json:"metadata"`
	Encoding   string       `json:"encoding"`
	Truncated  bool         `json:"truncated"`
	TotalLines int          `json:"totalLines"`
	Path       string       `json:"path"`
}

// FileEntry represents a file entry in a directory listing
type FileEntry struct {
	Path     string       `json:"path"`
	Name     string       `json:"name"`
	Metadata FileMetadata `json:"metadata"`
}

// SearchMatch represents a search match
type SearchMatch struct {
	Path    string        `json:"path"`
	Line    int           `json:"line"`
	Content string        `json:"content"`
	Context SearchContext `json:"context"`
}

// SearchContext represents lines around a match
type SearchContext struct {
	Before []string `json:"before"`
	After  []string `json:"after"`
}

// SearchResult represents search results
type SearchResult struct {
	Matches []SearchMatch `json:"matches"`
	Total   int           `json:"total"`
}

// WriteResult represents the result of a write operation
type WriteResult struct {
	Path         string `json:"path"`
	BytesWritten int64  `json:"bytesWritten"`
	Created      bool   `json:"created"` // true if file was created, false if overwritten
}

// CopyResult represents the result of a copy operation
type CopyResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	BytesCopied int64  `json:"bytesCopied"`
	IsDirectory bool   `json:"isDirectory"`
}

// MoveResult represents the result of a move operation
type MoveResult struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// DeleteResult represents the result of a delete operation
type DeleteResult struct {
	Path        string `json:"path"`
	IsDirectory bool   `json:"isDirectory"`
}

// ModifyResult represents the result of a modify operation
type ModifyResult struct {
	Path         string `json:"path"`
	Replacements int    `json:"replacements"`
	Modified     bool   `json:"modified"`
}

// ErrorCode represents file operation error codes
type ErrorCode string

const (
	ErrInvalidPath   ErrorCode = "INVALID_PATH"
	ErrFileNotFound  ErrorCode = "FILE_NOT_FOUND"
	ErrFileTooLarge  ErrorCode = "FILE_TOO_LARGE"
	ErrPermission    ErrorCode = "PERMISSION_DENIED"
	ErrAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrNotEmpty      ErrorCode = "NOT_EMPTY"
	ErrUnknown       ErrorCode = "UNKNOWN_ERROR"
)

// FileError represents a file operation error
type FileError struct {
	Code    ErrorCode
	Message string
	Path    string
}

func (e *FileError) Error() string {
	return fmt.Sprintf("%s: %s (path: %s)", e.Code, e.Message, e.Path)
}

// DefaultIgnorePatterns contains patterns to ignore by default
var DefaultIgnorePatterns = []string{
	".git",
	"node_modules",
	".vscode",
	".idea",
	"__pycache__",
	".DS_Store",
	"*.pyc",
	".env",
	"dist",
	"build",
	"coverage",
	".next",
	".nuxt",
	"vendor",
	".cache",
}

// GetMimeType returns the MIME type for a file
func GetMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	// Handle source code extensions first to avoid OS MIME type conflicts
	// (e.g., Windows returns "video/vnd.dlna.mpeg-tts" for .ts files)
	switch ext {
	case ".ts":
		return "text/typescript"
	case ".tsx":
		return "text/tsx"
	case ".jsx":
		return "text/jsx"
	case ".go":
		return "text/x-go"
	case ".rs":
		return "text/x-rust"
	case ".py":
		return "text/x-python"
	case ".rb":
		return "text/x-ruby"
	case ".java":
		return "text/x-java"
	case ".md":
		return "text/markdown"
	case ".yaml", ".yml":
		return "text/yaml"
	case ".toml":
		return "text/toml"
	case ".json":
		return "application/json"
	}

	// Fall back to system MIME type detection
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}

// GetFileMetadata returns metadata for a file
func GetFileMetadata(path string) (*FileMetadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileError{Code: ErrFileNotFound, Message: "File not found", Path: path}
		}
		if os.IsPermission(err) {
			return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
	}

	return &FileMetadata{
		Size:         info.Size(),
		MimeType:     GetMimeType(path),
		ModifiedTime: info.ModTime(),
		CreatedTime:  info.ModTime(), // Go doesn't have portable creation time
		IsDirectory:  info.IsDir(),
	}, nil
}

// ReadFile reads a file with optional size limit
func ReadFile(path string, maxSize int64) (*FileContent, error) {
	metadata, err := GetFileMetadata(path)
	if err != nil {
		return nil, err
	}

	if metadata.IsDirectory {
		return nil, &FileError{Code: ErrInvalidPath, Message: "Path is a directory", Path: path}
	}

	if maxSize > 0 && metadata.Size > maxSize {
		return nil, &FileError{Code: ErrFileTooLarge, Message: fmt.Sprintf("File exceeds max size of %d bytes", maxSize), Path: path}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
	}

	lines := strings.Count(string(content), "\n")
	if len(content) > 0 && content[len(content)-1] != '\n' {
		lines++
	}

	return &FileContent{
		Content:    string(content),
		Metadata:   *metadata,
		Encoding:   "utf-8",
		Truncated:  false,
		TotalLines: lines,
		Path:       path,
	}, nil
}

// ListFiles lists files in a directory
func ListFiles(dirPath string, recursive bool, fileTypes []string, includeHidden bool) ([]FileEntry, error) {
	metadata, err := GetFileMetadata(dirPath)
	if err != nil {
		return nil, err
	}

	if !metadata.IsDirectory {
		return nil, &FileError{Code: ErrInvalidPath, Message: "Path is not a directory", Path: dirPath}
	}

	var entries []FileEntry
	walkFn := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path
		relPath, _ := filepath.Rel(dirPath, path)
		if relPath == "." {
			return nil
		}

		name := d.Name()

		// Skip hidden files if not included
		if !includeHidden && strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored patterns
		for _, pattern := range DefaultIgnorePatterns {
			if matched, _ := doublestar.Match(pattern, name); matched {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Filter by file types
		if !d.IsDir() && len(fileTypes) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(name), ".")
			found := false
			for _, ft := range fileTypes {
				if strings.EqualFold(ext, ft) {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		entries = append(entries, FileEntry{
			Path: filepath.ToSlash(path),
			Name: name,
			Metadata: FileMetadata{
				Size:         info.Size(),
				MimeType:     GetMimeType(path),
				ModifiedTime: info.ModTime(),
				CreatedTime:  info.ModTime(),
				IsDirectory:  d.IsDir(),
			},
		})

		if !recursive && d.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	if recursive {
		err = filepath.WalkDir(dirPath, walkFn)
	} else {
		entries, err = readDirNonRecursive(dirPath, fileTypes, includeHidden)
	}

	if err != nil {
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: dirPath}
	}

	return entries, nil
}

func readDirNonRecursive(dirPath string, fileTypes []string, includeHidden bool) ([]FileEntry, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var entries []FileEntry
	for _, d := range dirEntries {
		name := d.Name()

		if !includeHidden && strings.HasPrefix(name, ".") {
			continue
		}

		// Skip ignored patterns
		skip := false
		for _, pattern := range DefaultIgnorePatterns {
			if matched, _ := doublestar.Match(pattern, name); matched {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Filter by file types
		if !d.IsDir() && len(fileTypes) > 0 {
			ext := strings.TrimPrefix(filepath.Ext(name), ".")
			found := false
			for _, ft := range fileTypes {
				if strings.EqualFold(ext, ft) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		info, err := d.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(dirPath, name)
		entries = append(entries, FileEntry{
			Path: filepath.ToSlash(fullPath),
			Name: name,
			Metadata: FileMetadata{
				Size:         info.Size(),
				MimeType:     GetMimeType(fullPath),
				ModifiedTime: info.ModTime(),
				CreatedTime:  info.ModTime(),
				IsDirectory:  d.IsDir(),
			},
		})
	}

	return entries, nil
}

// SearchFiles searches for a pattern in files
func SearchFiles(basePath string, pattern string, recursive bool, fileTypes []string, contextLines int, maxResults int) (*SearchResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &FileError{Code: ErrInvalidPath, Message: fmt.Sprintf("Invalid regex pattern: %s", err.Error()), Path: basePath}
	}

	entries, err := ListFiles(basePath, recursive, fileTypes, false)
	if err != nil {
		return nil, err
	}

	var matches []SearchMatch
	for _, entry := range entries {
		if entry.Metadata.IsDirectory {
			continue
		}

		fileMatches, err := searchInFile(entry.Path, re, contextLines)
		if err != nil {
			continue // Skip files that can't be read
		}

		matches = append(matches, fileMatches...)

		if maxResults > 0 && len(matches) >= maxResults {
			matches = matches[:maxResults]
			break
		}
	}

	return &SearchResult{
		Matches: matches,
		Total:   len(matches),
	}, nil
}

func searchInFile(path string, re *regexp.Regexp, contextLines int) ([]SearchMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []SearchMatch
	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	for i, line := range lines {
		if re.MatchString(line) {
			var before, after []string

			// Get context before
			start := i - contextLines
			if start < 0 {
				start = 0
			}
			before = lines[start:i]

			// Get context after
			end := i + contextLines + 1
			if end > len(lines) {
				end = len(lines)
			}
			if i+1 < len(lines) {
				after = lines[i+1 : end]
			}

			matches = append(matches, SearchMatch{
				Path:    filepath.ToSlash(path),
				Line:    i + 1,
				Content: line,
				Context: SearchContext{
					Before: before,
					After:  after,
				},
			})
		}
	}

	return matches, nil
}

// ReadDirectory reads all files in a directory and returns their contents
func ReadDirectory(dirPath string, recursive bool, fileTypes []string, maxSize int64) (map[string]*FileContent, error) {
	entries, err := ListFiles(dirPath, recursive, fileTypes, false)
	if err != nil {
		return nil, err
	}

	contents := make(map[string]*FileContent)
	for _, entry := range entries {
		if entry.Metadata.IsDirectory {
			continue
		}

		content, err := ReadFile(entry.Path, maxSize)
		if err != nil {
			continue // Skip files that can't be read
		}

		contents[entry.Path] = content
	}

	return contents, nil
}

// WriteFile creates a new file or overwrites an existing file with content
func WriteFile(path string, content string) (*WriteResult, error) {
	// Check if file exists to determine if we're creating or overwriting
	_, err := os.Stat(path)
	created := os.IsNotExist(err)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, &FileError{Code: ErrPermission, Message: fmt.Sprintf("Failed to create parent directory: %s", err.Error()), Path: path}
	}

	// Write the file
	data := []byte(content)
	if err := os.WriteFile(path, data, 0644); err != nil {
		if os.IsPermission(err) {
			return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
	}

	return &WriteResult{
		Path:         path,
		BytesWritten: int64(len(data)),
		Created:      created,
	}, nil
}

// CreateDirectory creates a new directory (including parent directories if needed)
func CreateDirectory(path string) error {
	// Check if already exists
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			// Directory already exists, this is fine
			return nil
		}
		return &FileError{Code: ErrAlreadyExists, Message: "Path exists but is not a directory", Path: path}
	}

	// Create the directory
	if err := os.MkdirAll(path, 0755); err != nil {
		if os.IsPermission(err) {
			return &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
		}
		return &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
	}

	return nil
}

// CopyFile copies a file or directory from source to destination
func CopyFile(source, destination string) (*CopyResult, error) {
	srcInfo, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileError{Code: ErrFileNotFound, Message: "Source not found", Path: source}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: source}
	}

	if srcInfo.IsDir() {
		return copyDirectory(source, destination)
	}

	return copyFileOnly(source, destination)
}

func copyFileOnly(source, destination string) (*CopyResult, error) {
	srcFile, err := os.Open(source)
	if err != nil {
		if os.IsPermission(err) {
			return nil, &FileError{Code: ErrPermission, Message: "Permission denied reading source", Path: source}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: source}
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, &FileError{Code: ErrPermission, Message: fmt.Sprintf("Failed to create destination directory: %s", err.Error()), Path: destination}
	}

	dstFile, err := os.Create(destination)
	if err != nil {
		if os.IsPermission(err) {
			return nil, &FileError{Code: ErrPermission, Message: "Permission denied writing destination", Path: destination}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: destination}
	}
	defer dstFile.Close()

	bytesCopied, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: destination}
	}

	// Copy file permissions
	srcInfo, _ := os.Stat(source)
	os.Chmod(destination, srcInfo.Mode())

	return &CopyResult{
		Source:      source,
		Destination: destination,
		BytesCopied: bytesCopied,
		IsDirectory: false,
	}, nil
}

func copyDirectory(source, destination string) (*CopyResult, error) {
	var totalBytes int64

	err := filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destination, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Copy file
		result, err := copyFileOnly(path, destPath)
		if err != nil {
			return err
		}
		totalBytes += result.BytesCopied

		return nil
	})

	if err != nil {
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: source}
	}

	return &CopyResult{
		Source:      source,
		Destination: destination,
		BytesCopied: totalBytes,
		IsDirectory: true,
	}, nil
}

// MoveFile moves or renames a file or directory
func MoveFile(source, destination string) (*MoveResult, error) {
	// Check source exists
	_, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileError{Code: ErrFileNotFound, Message: "Source not found", Path: source}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: source}
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destination)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, &FileError{Code: ErrPermission, Message: fmt.Sprintf("Failed to create destination directory: %s", err.Error()), Path: destination}
	}

	// Try atomic rename first (works if source and dest are on same filesystem)
	if err := os.Rename(source, destination); err != nil {
		// If rename fails (e.g., cross-device), try copy+delete
		srcInfo, _ := os.Stat(source)
		if srcInfo.IsDir() {
			_, copyErr := copyDirectory(source, destination)
			if copyErr != nil {
				return nil, copyErr
			}
		} else {
			_, copyErr := copyFileOnly(source, destination)
			if copyErr != nil {
				return nil, copyErr
			}
		}

		// Delete source after successful copy
		if err := os.RemoveAll(source); err != nil {
			return nil, &FileError{Code: ErrUnknown, Message: fmt.Sprintf("Copied but failed to delete source: %s", err.Error()), Path: source}
		}
	}

	return &MoveResult{
		Source:      source,
		Destination: destination,
	}, nil
}

// DeleteFile deletes a file or directory
func DeleteFile(path string, recursive bool) (*DeleteResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileError{Code: ErrFileNotFound, Message: "Path not found", Path: path}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
	}

	isDir := info.IsDir()

	if isDir && !recursive {
		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
		}
		if len(entries) > 0 {
			return nil, &FileError{Code: ErrNotEmpty, Message: "Directory is not empty. Use recursive=true to delete non-empty directories", Path: path}
		}
		// Remove empty directory
		if err := os.Remove(path); err != nil {
			if os.IsPermission(err) {
				return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
			}
			return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
		}
	} else if isDir && recursive {
		// Remove directory recursively
		if err := os.RemoveAll(path); err != nil {
			if os.IsPermission(err) {
				return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
			}
			return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
		}
	} else {
		// Remove file
		if err := os.Remove(path); err != nil {
			if os.IsPermission(err) {
				return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
			}
			return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
		}
	}

	return &DeleteResult{
		Path:        path,
		IsDirectory: isDir,
	}, nil
}

// ModifyFile performs find and replace operations on a file
func ModifyFile(path string, find string, replace string, allOccurrences bool, useRegex bool) (*ModifyResult, error) {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &FileError{Code: ErrFileNotFound, Message: "File not found", Path: path}
		}
		if os.IsPermission(err) {
			return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
		}
		return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
	}

	originalContent := string(content)
	var newContent string
	var replacements int

	if useRegex {
		re, err := regexp.Compile(find)
		if err != nil {
			return nil, &FileError{Code: ErrInvalidPath, Message: fmt.Sprintf("Invalid regex pattern: %s", err.Error()), Path: path}
		}

		if allOccurrences {
			matches := re.FindAllStringIndex(originalContent, -1)
			replacements = len(matches)
			newContent = re.ReplaceAllString(originalContent, replace)
		} else {
			if re.MatchString(originalContent) {
				replacements = 1
				newContent = re.ReplaceAllStringFunc(originalContent, func(match string) string {
					if replacements == 1 {
						replacements = 0 // Mark as replaced
						return re.ReplaceAllString(match, replace)
					}
					return match
				})
				replacements = 1 // Reset to correct count
			} else {
				newContent = originalContent
				replacements = 0
			}
		}
	} else {
		if allOccurrences {
			replacements = strings.Count(originalContent, find)
			newContent = strings.ReplaceAll(originalContent, find, replace)
		} else {
			if strings.Contains(originalContent, find) {
				replacements = 1
				newContent = strings.Replace(originalContent, find, replace, 1)
			} else {
				newContent = originalContent
				replacements = 0
			}
		}
	}

	modified := newContent != originalContent

	if modified {
		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
			if os.IsPermission(err) {
				return nil, &FileError{Code: ErrPermission, Message: "Permission denied", Path: path}
			}
			return nil, &FileError{Code: ErrUnknown, Message: err.Error(), Path: path}
		}
	}

	return &ModifyResult{
		Path:         path,
		Replacements: replacements,
		Modified:     modified,
	}, nil
}
