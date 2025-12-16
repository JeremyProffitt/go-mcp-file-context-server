package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetMimeType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"test.go", "text/x-go"},
		{"test.ts", "text/typescript"},
		{"test.py", "text/x-python"},
		{"test.js", "text/javascript"},
		{"test.json", "application/json"},
		{"test.md", "text/markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetMimeType(tt.path)
			if result != tt.expected {
				t.Errorf("GetMimeType(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!\nLine 2\nLine 3"

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading the file
	result, err := ReadFile(testFile, 0)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if result.Content != content {
		t.Errorf("Content mismatch: got %q, want %q", result.Content, content)
	}

	if result.TotalLines != 3 {
		t.Errorf("TotalLines = %d, want 3", result.TotalLines)
	}

	if result.Metadata.Size != int64(len(content)) {
		t.Errorf("Size = %d, want %d", result.Metadata.Size, len(content))
	}
}

func TestReadFileNotFound(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.txt", 0)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	fileErr, ok := err.(*FileError)
	if !ok {
		t.Errorf("Expected FileError, got %T", err)
	}

	if fileErr.Code != ErrFileNotFound {
		t.Errorf("Expected ErrFileNotFound, got %s", fileErr.Code)
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{"test1.go", "test2.ts", "test3.py"}
	for _, f := range testFiles {
		err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create a subdirectory with a file
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	err = os.WriteFile(filepath.Join(subDir, "nested.go"), []byte("nested"), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// Test non-recursive listing
	entries, err := ListFiles(tmpDir, false, nil, false)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	// Should include files and subdir but not nested file
	fileCount := 0
	for _, e := range entries {
		if !e.Metadata.IsDirectory {
			fileCount++
		}
	}
	if fileCount != 3 {
		t.Errorf("Non-recursive: expected 3 files, got %d", fileCount)
	}

	// Test recursive listing
	entries, err = ListFiles(tmpDir, true, nil, false)
	if err != nil {
		t.Fatalf("ListFiles recursive failed: %v", err)
	}

	fileCount = 0
	for _, e := range entries {
		if !e.Metadata.IsDirectory {
			fileCount++
		}
	}
	if fileCount != 4 {
		t.Errorf("Recursive: expected 4 files, got %d", fileCount)
	}

	// Test file type filter
	entries, err = ListFiles(tmpDir, true, []string{"go"}, false)
	if err != nil {
		t.Fatalf("ListFiles with filter failed: %v", err)
	}

	fileCount = 0
	for _, e := range entries {
		if !e.Metadata.IsDirectory {
			fileCount++
		}
	}
	if fileCount != 2 {
		t.Errorf("Filtered: expected 2 .go files, got %d", fileCount)
	}
}

func TestSearchFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with content
	files := map[string]string{
		"test1.go": "package main\n\nfunc Hello() {\n\treturn\n}",
		"test2.go": "package main\n\nfunc World() {\n\treturn\n}",
		"test3.py": "def hello():\n    pass",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Search for "func"
	results, err := SearchFiles(tmpDir, "func", true, nil, 1, 100)
	if err != nil {
		t.Fatalf("SearchFiles failed: %v", err)
	}

	if results.Total != 2 {
		t.Errorf("Expected 2 matches for 'func', got %d", results.Total)
	}

	// Search with file type filter
	results, err = SearchFiles(tmpDir, "def", true, []string{"py"}, 1, 100)
	if err != nil {
		t.Fatalf("SearchFiles with filter failed: %v", err)
	}

	if results.Total != 1 {
		t.Errorf("Expected 1 match for 'def' in .py files, got %d", results.Total)
	}
}

func TestGetFileMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	err := os.WriteFile(testFile, []byte("package main"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	meta, err := GetFileMetadata(testFile)
	if err != nil {
		t.Fatalf("GetFileMetadata failed: %v", err)
	}

	if meta.IsDirectory {
		t.Error("Expected file, got directory")
	}

	if meta.Size != 12 {
		t.Errorf("Expected size 12, got %d", meta.Size)
	}

	if meta.MimeType != "text/x-go" {
		t.Errorf("Expected mime type text/x-go, got %s", meta.MimeType)
	}
}
