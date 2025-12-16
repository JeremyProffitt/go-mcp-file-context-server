package analysis

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/files"
)

// CodeAnalysis represents code analysis results
type CodeAnalysis struct {
	Definitions []string `json:"definitions,omitempty"`
	Imports     []string `json:"imports,omitempty"`
	Complexity  int      `json:"complexity"`
}

// QualityMetrics represents code quality metrics
type QualityMetrics struct {
	TotalLines      int      `json:"totalLines"`
	NonEmptyLines   int      `json:"nonEmptyLines"`
	CommentLines    int      `json:"commentLines"`
	DuplicateLines  int      `json:"duplicateLines"`
	LongLines       int      `json:"longLines"`
	ComplexFuncs    int      `json:"complexFunctions"`
	AvgComplexity   float64  `json:"avgComplexity"`
	MaxComplexity   int      `json:"maxComplexity"`
	FilesAnalyzed   int      `json:"filesAnalyzed"`
}

// FileAnalysis represents analysis results for a single file
type FileAnalysis struct {
	Path           string       `json:"path"`
	Analysis       CodeAnalysis `json:"analysis"`
	QualityMetrics QualityMetrics `json:"qualityMetrics,omitempty"`
}

// Outline represents a code outline
type Outline struct {
	Path      string         `json:"path"`
	Language  string         `json:"language"`
	Classes   []ClassOutline `json:"classes,omitempty"`
	Functions []FuncOutline  `json:"functions,omitempty"`
	Imports   []string       `json:"imports,omitempty"`
	Exports   []string       `json:"exports,omitempty"`
}

// ClassOutline represents a class in the outline
type ClassOutline struct {
	Name    string        `json:"name"`
	Line    int           `json:"line"`
	Methods []FuncOutline `json:"methods,omitempty"`
}

// FuncOutline represents a function in the outline
type FuncOutline struct {
	Name       string `json:"name"`
	Line       int    `json:"line"`
	Params     string `json:"params,omitempty"`
	ReturnType string `json:"returnType,omitempty"`
}

// Language patterns
var (
	// Import patterns
	importPatterns = map[string]*regexp.Regexp{
		"go":         regexp.MustCompile(`(?m)^import\s+(?:\(([^)]+)\)|"([^"]+)")`),
		"typescript": regexp.MustCompile(`(?m)^import\s+.*?from\s+['"]([^'"]+)['"]`),
		"javascript": regexp.MustCompile(`(?m)^(?:import\s+.*?from\s+['"]([^'"]+)['"]|const\s+\w+\s*=\s*require\(['"]([^'"]+)['"]\))`),
		"python":     regexp.MustCompile(`(?m)^(?:import\s+(\S+)|from\s+(\S+)\s+import)`),
		"java":       regexp.MustCompile(`(?m)^import\s+([\w.]+);`),
		"rust":       regexp.MustCompile(`(?m)^use\s+([\w:]+)`),
	}

	// Function/method patterns
	funcPatterns = map[string]*regexp.Regexp{
		"go":         regexp.MustCompile(`(?m)^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(([^)]*)\)(?:\s*\(?([^){\n]*)\)?)?`),
		"typescript": regexp.MustCompile(`(?m)(?:^|\s)(?:async\s+)?(?:function\s+)?(\w+)\s*(?:<[^>]+>)?\s*\(([^)]*)\)(?:\s*:\s*([^\s{]+))?`),
		"javascript": regexp.MustCompile(`(?m)(?:^|\s)(?:async\s+)?(?:function\s+)?(\w+)\s*\(([^)]*)\)`),
		"python":     regexp.MustCompile(`(?m)^\s*def\s+(\w+)\s*\(([^)]*)\)(?:\s*->\s*(\S+))?`),
		"java":       regexp.MustCompile(`(?m)(?:public|private|protected)?\s*(?:static)?\s*(?:\w+(?:<[^>]+>)?)\s+(\w+)\s*\(([^)]*)\)`),
		"rust":       regexp.MustCompile(`(?m)^(?:pub\s+)?fn\s+(\w+)(?:<[^>]+>)?\s*\(([^)]*)\)(?:\s*->\s*(\S+))?`),
	}

	// Class patterns
	classPatterns = map[string]*regexp.Regexp{
		"go":         regexp.MustCompile(`(?m)^type\s+(\w+)\s+struct`),
		"typescript": regexp.MustCompile(`(?m)^(?:export\s+)?class\s+(\w+)`),
		"javascript": regexp.MustCompile(`(?m)^class\s+(\w+)`),
		"python":     regexp.MustCompile(`(?m)^class\s+(\w+)`),
		"java":       regexp.MustCompile(`(?m)(?:public|private)?\s*class\s+(\w+)`),
		"rust":       regexp.MustCompile(`(?m)^(?:pub\s+)?struct\s+(\w+)`),
	}

	// Complexity patterns (control flow)
	complexityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\bif\b`),
		regexp.MustCompile(`\belse\s+if\b`),
		regexp.MustCompile(`\bfor\b`),
		regexp.MustCompile(`\bwhile\b`),
		regexp.MustCompile(`\bswitch\b`),
		regexp.MustCompile(`\bcase\b`),
		regexp.MustCompile(`\bcatch\b`),
		regexp.MustCompile(`\b\?\s*[^:]+:`), // ternary
		regexp.MustCompile(`&&`),
		regexp.MustCompile(`\|\|`),
	}
)

// GetLanguage determines the language from file extension
func GetLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs":
		return "javascript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	default:
		return "unknown"
	}
}

// AnalyzeFile performs code analysis on a file
func AnalyzeFile(path string) (*FileAnalysis, error) {
	content, err := files.ReadFile(path, 0)
	if err != nil {
		return nil, err
	}

	lang := GetLanguage(path)
	analysis := &CodeAnalysis{
		Complexity: calculateComplexity(content.Content),
	}

	// Extract imports
	if pattern, ok := importPatterns[lang]; ok {
		matches := pattern.FindAllStringSubmatch(content.Content, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					analysis.Imports = append(analysis.Imports, strings.TrimSpace(match[i]))
				}
			}
		}
	}

	// Extract definitions (functions and classes)
	if pattern, ok := funcPatterns[lang]; ok {
		matches := pattern.FindAllStringSubmatch(content.Content, -1)
		for _, match := range matches {
			if len(match) > 1 && match[1] != "" {
				analysis.Definitions = append(analysis.Definitions, match[1])
			}
		}
	}

	if pattern, ok := classPatterns[lang]; ok {
		matches := pattern.FindAllStringSubmatch(content.Content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				analysis.Definitions = append(analysis.Definitions, match[1])
			}
		}
	}

	quality := calculateQuality(content.Content)

	return &FileAnalysis{
		Path:           path,
		Analysis:       *analysis,
		QualityMetrics: quality,
	}, nil
}

// GenerateOutline generates a code outline for a file
func GenerateOutline(path string) (*Outline, error) {
	content, err := files.ReadFile(path, 0)
	if err != nil {
		return nil, err
	}

	lang := GetLanguage(path)
	lines := strings.Split(content.Content, "\n")

	outline := &Outline{
		Path:     path,
		Language: lang,
	}

	// Extract imports
	if pattern, ok := importPatterns[lang]; ok {
		matches := pattern.FindAllStringSubmatch(content.Content, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					outline.Imports = append(outline.Imports, strings.TrimSpace(match[i]))
				}
			}
		}
	}

	// Extract classes with line numbers
	if pattern, ok := classPatterns[lang]; ok {
		for i, line := range lines {
			if matches := pattern.FindStringSubmatch(line); matches != nil && len(matches) > 1 {
				outline.Classes = append(outline.Classes, ClassOutline{
					Name: matches[1],
					Line: i + 1,
				})
			}
		}
	}

	// Extract functions with line numbers
	if pattern, ok := funcPatterns[lang]; ok {
		for i, line := range lines {
			if matches := pattern.FindStringSubmatch(line); matches != nil && len(matches) > 1 && matches[1] != "" {
				fn := FuncOutline{
					Name: matches[1],
					Line: i + 1,
				}
				if len(matches) > 2 {
					fn.Params = matches[2]
				}
				if len(matches) > 3 {
					fn.ReturnType = matches[3]
				}
				outline.Functions = append(outline.Functions, fn)
			}
		}
	}

	return outline, nil
}

func calculateComplexity(content string) int {
	complexity := 1 // Base complexity

	for _, pattern := range complexityPatterns {
		matches := pattern.FindAllString(content, -1)
		complexity += len(matches)
	}

	return complexity
}

func calculateQuality(content string) QualityMetrics {
	lines := strings.Split(content, "\n")
	metrics := QualityMetrics{
		TotalLines: len(lines),
	}

	lineCount := make(map[string]int)
	commentPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*//`),
		regexp.MustCompile(`^\s*#`),
		regexp.MustCompile(`^\s*/\*`),
		regexp.MustCompile(`^\s*\*`),
		regexp.MustCompile(`^\s*'''`),
		regexp.MustCompile(`^\s*"""`),
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Count non-empty lines
		if trimmed != "" {
			metrics.NonEmptyLines++
		}

		// Count comment lines
		for _, pattern := range commentPatterns {
			if pattern.MatchString(line) {
				metrics.CommentLines++
				break
			}
		}

		// Count long lines (>100 chars)
		if len(line) > 100 {
			metrics.LongLines++
		}

		// Track duplicates
		if trimmed != "" && len(trimmed) > 10 {
			lineCount[trimmed]++
		}
	}

	// Count duplicate lines
	for _, count := range lineCount {
		if count > 1 {
			metrics.DuplicateLines += count - 1
		}
	}

	return metrics
}

// AnalyzeDirectory analyzes all files in a directory
func AnalyzeDirectory(dirPath string, recursive bool, fileTypes []string) ([]FileAnalysis, *QualityMetrics, error) {
	entries, err := files.ListFiles(dirPath, recursive, fileTypes, false)
	if err != nil {
		return nil, nil, err
	}

	var analyses []FileAnalysis
	aggregateMetrics := &QualityMetrics{}
	var totalComplexity int

	for _, entry := range entries {
		if entry.Metadata.IsDirectory {
			continue
		}

		analysis, err := AnalyzeFile(entry.Path)
		if err != nil {
			continue
		}

		analyses = append(analyses, *analysis)

		// Aggregate metrics
		aggregateMetrics.TotalLines += analysis.QualityMetrics.TotalLines
		aggregateMetrics.NonEmptyLines += analysis.QualityMetrics.NonEmptyLines
		aggregateMetrics.CommentLines += analysis.QualityMetrics.CommentLines
		aggregateMetrics.DuplicateLines += analysis.QualityMetrics.DuplicateLines
		aggregateMetrics.LongLines += analysis.QualityMetrics.LongLines
		aggregateMetrics.FilesAnalyzed++
		totalComplexity += analysis.Analysis.Complexity

		if analysis.Analysis.Complexity > aggregateMetrics.MaxComplexity {
			aggregateMetrics.MaxComplexity = analysis.Analysis.Complexity
		}

		if analysis.Analysis.Complexity > 10 {
			aggregateMetrics.ComplexFuncs++
		}
	}

	if aggregateMetrics.FilesAnalyzed > 0 {
		aggregateMetrics.AvgComplexity = float64(totalComplexity) / float64(aggregateMetrics.FilesAnalyzed)
	}

	return analyses, aggregateMetrics, nil
}

// GetChunkCount calculates the number of chunks for a file
func GetChunkCount(path string, chunkSize int64) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	if info.IsDir() {
		// For directories, we need to calculate total content size
		entries, err := files.ListFiles(path, true, nil, false)
		if err != nil {
			return 0, err
		}

		var totalSize int64
		for _, entry := range entries {
			if !entry.Metadata.IsDirectory {
				totalSize += entry.Metadata.Size
			}
		}

		return int((totalSize + chunkSize - 1) / chunkSize), nil
	}

	return int((info.Size() + chunkSize - 1) / chunkSize), nil
}

// ReadChunk reads a specific chunk of content
func ReadChunk(path string, chunkNumber int, chunkSize int64) (string, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	totalChunks := int((info.Size() + chunkSize - 1) / chunkSize)
	if chunkNumber >= totalChunks {
		return "", totalChunks, nil
	}

	offset := int64(chunkNumber) * chunkSize
	_, err = file.Seek(offset, 0)
	if err != nil {
		return "", totalChunks, err
	}

	buf := make([]byte, chunkSize)
	n, err := file.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return "", totalChunks, err
	}

	return string(buf[:n]), totalChunks, nil
}

// GetFolderStructure returns a tree representation of the folder structure
func GetFolderStructure(dirPath string, maxDepth int) (string, error) {
	var builder strings.Builder

	err := walkDir(dirPath, "", 0, maxDepth, &builder)
	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func walkDir(path string, prefix string, depth int, maxDepth int, builder *strings.Builder) error {
	if maxDepth > 0 && depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Filter out ignored patterns
	var filtered []os.DirEntry
	for _, entry := range entries {
		skip := false
		for _, pattern := range files.DefaultIgnorePatterns {
			if matched, _ := filepath.Match(pattern, entry.Name()); matched {
				skip = true
				break
			}
		}
		if strings.HasPrefix(entry.Name(), ".") {
			skip = true
		}
		if !skip {
			filtered = append(filtered, entry)
		}
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		if isLast {
			connector = "└── "
		}

		builder.WriteString(prefix + connector + entry.Name() + "\n")

		if entry.IsDir() {
			newPrefix := prefix + "│   "
			if isLast {
				newPrefix = prefix + "    "
			}
			walkDir(filepath.Join(path, entry.Name()), newPrefix, depth+1, maxDepth, builder)
		}
	}

	return nil
}

// CountLines counts lines in a file efficiently
func CountLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}

	return count, scanner.Err()
}
