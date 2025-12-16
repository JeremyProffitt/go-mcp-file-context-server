package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"component.tsx", "typescript"},
		{"script.js", "javascript"},
		{"app.py", "python"},
		{"Main.java", "java"},
		{"lib.rs", "rust"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetLanguage(tt.path)
			if result != tt.expected {
				t.Errorf("GetLanguage(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestAnalyzeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go test file
	goCode := `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		fmt.Println(os.Args[1])
	} else {
		fmt.Println("Hello")
	}
}

func helper() string {
	return "test"
}
`
	testFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(testFile, []byte(goCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	analysis, err := AnalyzeFile(testFile)
	if err != nil {
		t.Fatalf("AnalyzeFile failed: %v", err)
	}

	// Check complexity (should be > 1 due to if/else)
	if analysis.Analysis.Complexity <= 1 {
		t.Errorf("Expected complexity > 1, got %d", analysis.Analysis.Complexity)
	}

	// Check imports were extracted
	if len(analysis.Analysis.Imports) == 0 {
		t.Error("Expected imports to be extracted")
	}

	// Check definitions were extracted
	if len(analysis.Analysis.Definitions) < 2 {
		t.Errorf("Expected at least 2 definitions (main, helper), got %d", len(analysis.Analysis.Definitions))
	}
}

func TestGenerateOutline(t *testing.T) {
	tmpDir := t.TempDir()

	// Test Go file
	goCode := `package main

import "fmt"

type Server struct {
	port int
}

func (s *Server) Start() {
	fmt.Println("Starting")
}

func main() {
	s := &Server{port: 8080}
	s.Start()
}
`
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, []byte(goCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outline, err := GenerateOutline(goFile)
	if err != nil {
		t.Fatalf("GenerateOutline failed: %v", err)
	}

	if outline.Language != "go" {
		t.Errorf("Expected language 'go', got %s", outline.Language)
	}

	// Should have at least one class (struct)
	if len(outline.Classes) < 1 {
		t.Error("Expected at least one class (Server struct)")
	}

	// Should have functions
	if len(outline.Functions) < 2 {
		t.Errorf("Expected at least 2 functions, got %d", len(outline.Functions))
	}
}

func TestGenerateOutlinePython(t *testing.T) {
	tmpDir := t.TempDir()

	pythonCode := `import os
from typing import Optional

class MyClass:
    def __init__(self, name: str):
        self.name = name

    def greet(self) -> str:
        return f"Hello, {self.name}"

def main():
    obj = MyClass("World")
    print(obj.greet())
`
	pyFile := filepath.Join(tmpDir, "main.py")
	err := os.WriteFile(pyFile, []byte(pythonCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	outline, err := GenerateOutline(pyFile)
	if err != nil {
		t.Fatalf("GenerateOutline failed: %v", err)
	}

	if outline.Language != "python" {
		t.Errorf("Expected language 'python', got %s", outline.Language)
	}

	if len(outline.Classes) < 1 {
		t.Error("Expected at least one class")
	}

	// Should have functions including class methods
	if len(outline.Functions) < 3 {
		t.Errorf("Expected at least 3 functions, got %d", len(outline.Functions))
	}
}

func TestGetFolderStructure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory structure
	dirs := []string{"src", "src/components", "pkg", "docs"}
	for _, d := range dirs {
		err := os.MkdirAll(filepath.Join(tmpDir, d), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	files := []string{
		"README.md",
		"main.go",
		"src/app.ts",
		"src/components/button.tsx",
		"pkg/utils.go",
	}
	for _, f := range files {
		err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	structure, err := GetFolderStructure(tmpDir, 5)
	if err != nil {
		t.Fatalf("GetFolderStructure failed: %v", err)
	}

	// Check that key items are present
	expectedItems := []string{"README.md", "main.go", "src", "pkg", "docs"}
	for _, item := range expectedItems {
		if !strings.Contains(structure, item) {
			t.Errorf("Expected structure to contain %s", item)
		}
	}
}

func TestGetChunkCount(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with known size
	content := make([]byte, 1000)
	for i := range content {
		content[i] = 'a'
	}

	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 1000 bytes with chunk size of 256 should give 4 chunks
	count, err := GetChunkCount(testFile, 256)
	if err != nil {
		t.Fatalf("GetChunkCount failed: %v", err)
	}

	if count != 4 {
		t.Errorf("Expected 4 chunks, got %d", count)
	}
}

func TestReadChunk(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with sequential content
	content := "0123456789ABCDEFGHIJ"
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read first chunk
	chunk, total, err := ReadChunk(testFile, 0, 10)
	if err != nil {
		t.Fatalf("ReadChunk failed: %v", err)
	}

	if total != 2 {
		t.Errorf("Expected 2 total chunks, got %d", total)
	}

	if chunk != "0123456789" {
		t.Errorf("Expected '0123456789', got %s", chunk)
	}

	// Read second chunk
	chunk, _, err = ReadChunk(testFile, 1, 10)
	if err != nil {
		t.Fatalf("ReadChunk second failed: %v", err)
	}

	if chunk != "ABCDEFGHIJ" {
		t.Errorf("Expected 'ABCDEFGHIJ', got %s", chunk)
	}
}

func TestCountLines(t *testing.T) {
	tmpDir := t.TempDir()

	content := "line1\nline2\nline3\nline4\nline5"
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	count, err := CountLines(testFile)
	if err != nil {
		t.Fatalf("CountLines failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 lines, got %d", count)
	}
}
