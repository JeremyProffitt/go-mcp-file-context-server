package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

	// Check prefix to handle charset suffix variations (e.g., "text/x-go; charset=utf-8")
	if !strings.HasPrefix(meta.MimeType, "text/x-go") {
		t.Errorf("Expected mime type prefix text/x-go, got %s", meta.MimeType)
	}
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating a new file
	newFile := filepath.Join(tmpDir, "newfile.txt")
	content := "Hello, World!"

	result, err := WriteFile(newFile, content)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if !result.Created {
		t.Error("Expected Created to be true for new file")
	}

	if result.BytesWritten != int64(len(content)) {
		t.Errorf("Expected %d bytes written, got %d", len(content), result.BytesWritten)
	}

	// Verify file content
	data, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(data), content)
	}

	// Test overwriting existing file
	newContent := "Updated content"
	result, err = WriteFile(newFile, newContent)
	if err != nil {
		t.Fatalf("WriteFile overwrite failed: %v", err)
	}

	if result.Created {
		t.Error("Expected Created to be false for overwritten file")
	}

	// Test creating file in nested directory
	nestedFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")
	result, err = WriteFile(nestedFile, "nested content")
	if err != nil {
		t.Fatalf("WriteFile nested failed: %v", err)
	}

	if !result.Created {
		t.Error("Expected Created to be true for new nested file")
	}
}

func TestCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating a new directory
	newDir := filepath.Join(tmpDir, "newdir")
	err := CreateDirectory(newDir)
	if err != nil {
		t.Fatalf("CreateDirectory failed: %v", err)
	}

	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("Created directory does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}

	// Test creating nested directories
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	err = CreateDirectory(nestedDir)
	if err != nil {
		t.Fatalf("CreateDirectory nested failed: %v", err)
	}

	// Test creating directory that already exists (should not error)
	err = CreateDirectory(newDir)
	if err != nil {
		t.Errorf("CreateDirectory on existing dir should not error: %v", err)
	}

	// Test creating directory where file exists
	testFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	err = CreateDirectory(testFile)
	if err == nil {
		t.Error("Expected error when creating directory where file exists")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	content := "Copy me!"
	os.WriteFile(srcFile, []byte(content), 0644)

	// Test copying file
	dstFile := filepath.Join(tmpDir, "dest.txt")
	result, err := CopyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	if result.IsDirectory {
		t.Error("Expected IsDirectory to be false")
	}

	if result.BytesCopied != int64(len(content)) {
		t.Errorf("Expected %d bytes copied, got %d", len(content), result.BytesCopied)
	}

	// Verify destination content
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(data), content)
	}

	// Test copying directory
	srcDir := filepath.Join(tmpDir, "srcdir")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("file1"), 0644)
	os.WriteFile(filepath.Join(srcDir, "file2.txt"), []byte("file2"), 0644)
	os.Mkdir(filepath.Join(srcDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("nested"), 0644)

	dstDir := filepath.Join(tmpDir, "dstdir")
	result, err = CopyFile(srcDir, dstDir)
	if err != nil {
		t.Fatalf("CopyFile directory failed: %v", err)
	}

	if !result.IsDirectory {
		t.Error("Expected IsDirectory to be true")
	}

	// Verify copied files exist
	if _, err := os.Stat(filepath.Join(dstDir, "file1.txt")); err != nil {
		t.Error("file1.txt not copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "subdir", "nested.txt")); err != nil {
		t.Error("nested.txt not copied")
	}

	// Test copying non-existent file
	_, err = CopyFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest"))
	if err == nil {
		t.Error("Expected error when copying non-existent file")
	}
}

func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "source.txt")
	content := "Move me!"
	os.WriteFile(srcFile, []byte(content), 0644)

	// Test moving file
	dstFile := filepath.Join(tmpDir, "dest.txt")
	result, err := MoveFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("MoveFile failed: %v", err)
	}

	if result.Source != srcFile || result.Destination != dstFile {
		t.Error("Result paths don't match")
	}

	// Verify source no longer exists
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("Source file should not exist after move")
	}

	// Verify destination content
	data, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(data), content)
	}

	// Test moving non-existent file
	_, err = MoveFile(filepath.Join(tmpDir, "nonexistent"), filepath.Join(tmpDir, "dest2"))
	if err == nil {
		t.Error("Expected error when moving non-existent file")
	}
}

func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "delete_me.txt")
	os.WriteFile(testFile, []byte("delete"), 0644)

	// Test deleting file
	result, err := DeleteFile(testFile, false)
	if err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	if result.IsDirectory {
		t.Error("Expected IsDirectory to be false")
	}

	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("File should not exist after delete")
	}

	// Create directory with files
	testDir := filepath.Join(tmpDir, "delete_dir")
	os.Mkdir(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)

	// Test deleting non-empty directory without recursive flag
	_, err = DeleteFile(testDir, false)
	if err == nil {
		t.Error("Expected error when deleting non-empty directory without recursive")
	}

	// Test deleting non-empty directory with recursive flag
	result, err = DeleteFile(testDir, true)
	if err != nil {
		t.Fatalf("DeleteFile recursive failed: %v", err)
	}

	if !result.IsDirectory {
		t.Error("Expected IsDirectory to be true")
	}

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("Directory should not exist after recursive delete")
	}

	// Test deleting empty directory
	emptyDir := filepath.Join(tmpDir, "empty_dir")
	os.Mkdir(emptyDir, 0755)
	result, err = DeleteFile(emptyDir, false)
	if err != nil {
		t.Fatalf("DeleteFile empty dir failed: %v", err)
	}

	// Test deleting non-existent file
	_, err = DeleteFile(filepath.Join(tmpDir, "nonexistent"), false)
	if err == nil {
		t.Error("Expected error when deleting non-existent file")
	}
}

func TestModifyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "modify.txt")
	content := "Hello World, Hello Universe, Hello Galaxy"
	os.WriteFile(testFile, []byte(content), 0644)

	// Test single replacement
	result, err := ModifyFile(testFile, "Hello", "Hi", false, false)
	if err != nil {
		t.Fatalf("ModifyFile failed: %v", err)
	}

	if result.Replacements != 1 {
		t.Errorf("Expected 1 replacement, got %d", result.Replacements)
	}

	if !result.Modified {
		t.Error("Expected Modified to be true")
	}

	data, _ := os.ReadFile(testFile)
	expected := "Hi World, Hello Universe, Hello Galaxy"
	if string(data) != expected {
		t.Errorf("Content mismatch: got %q, want %q", string(data), expected)
	}

	// Test all occurrences replacement
	result, err = ModifyFile(testFile, "Hello", "Hi", true, false)
	if err != nil {
		t.Fatalf("ModifyFile all occurrences failed: %v", err)
	}

	if result.Replacements != 2 {
		t.Errorf("Expected 2 replacements, got %d", result.Replacements)
	}

	data, _ = os.ReadFile(testFile)
	expected = "Hi World, Hi Universe, Hi Galaxy"
	if string(data) != expected {
		t.Errorf("Content mismatch: got %q, want %q", string(data), expected)
	}

	// Test regex replacement
	testFile2 := filepath.Join(tmpDir, "regex.txt")
	os.WriteFile(testFile2, []byte("func foo() {}\nfunc bar() {}"), 0644)

	result, err = ModifyFile(testFile2, `func (\w+)\(\)`, "function $1()", true, true)
	if err != nil {
		t.Fatalf("ModifyFile regex failed: %v", err)
	}

	if result.Replacements != 2 {
		t.Errorf("Expected 2 regex replacements, got %d", result.Replacements)
	}

	data, _ = os.ReadFile(testFile2)
	expected = "function foo() {}\nfunction bar() {}"
	if string(data) != expected {
		t.Errorf("Content mismatch: got %q, want %q", string(data), expected)
	}

	// Test no match
	result, err = ModifyFile(testFile, "nonexistent", "replacement", true, false)
	if err != nil {
		t.Fatalf("ModifyFile no match failed: %v", err)
	}

	if result.Modified {
		t.Error("Expected Modified to be false when no matches")
	}

	if result.Replacements != 0 {
		t.Errorf("Expected 0 replacements, got %d", result.Replacements)
	}

	// Test modifying non-existent file
	_, err = ModifyFile(filepath.Join(tmpDir, "nonexistent"), "a", "b", true, false)
	if err == nil {
		t.Error("Expected error when modifying non-existent file")
	}
}
