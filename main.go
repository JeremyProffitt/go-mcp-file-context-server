package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/analysis"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/cache"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/files"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/mcp"
)

const (
	Version         = "1.0.0"
	DefaultMaxSize  = 10 * 1024 * 1024  // 10MB
	DefaultCacheSize = 500
	DefaultCacheTTL  = 5 * time.Minute
	DefaultChunkSize = 64 * 1024        // 64KB
)

var fileCache *cache.Cache

func main() {
	// Initialize cache
	var err error
	fileCache, err = cache.NewCache(DefaultCacheSize, DefaultCacheTTL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize cache: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server
	server := mcp.NewServer("file-context-server", Version)

	// Register tools
	registerTools(server)

	// Run the server
	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func registerTools(server *mcp.Server) {
	// list_context_files tool
	server.RegisterTool(mcp.Tool{
		Name:        "list_context_files",
		Description: "Lists files in a directory with detailed metadata",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Directory path to list files from",
				},
				"recursive": {
					Type:        "boolean",
					Description: "Include subdirectories",
					Default:     true,
				},
				"includeHidden": {
					Type:        "boolean",
					Description: "Include hidden files",
					Default:     false,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter by file extensions (without dots)",
					Items:       &mcp.Property{Type: "string"},
				},
			},
			Required: []string{"path"},
		},
	}, handleListContextFiles)

	// read_context tool
	server.RegisterTool(mcp.Tool{
		Name:        "read_context",
		Description: "Reads file or directory contents with metadata and caching",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "File or directory path to read",
				},
				"maxSize": {
					Type:        "number",
					Description: "Maximum file size in bytes",
					Default:     float64(DefaultMaxSize),
				},
				"encoding": {
					Type:        "string",
					Description: "Character encoding",
					Default:     "utf8",
				},
				"recursive": {
					Type:        "boolean",
					Description: "Include subdirectories for directory paths",
					Default:     true,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter by file extensions (without dots)",
					Items:       &mcp.Property{Type: "string"},
				},
				"chunkNumber": {
					Type:        "number",
					Description: "Chunk number for large files (0-indexed)",
					Default:     float64(0),
				},
			},
			Required: []string{"path"},
		},
	}, handleReadContext)

	// search_context tool
	server.RegisterTool(mcp.Tool{
		Name:        "search_context",
		Description: "Searches for patterns in files with context",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"pattern": {
					Type:        "string",
					Description: "Regex pattern to search for",
				},
				"path": {
					Type:        "string",
					Description: "Directory to search in",
				},
				"recursive": {
					Type:        "boolean",
					Description: "Search in subdirectories",
					Default:     true,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter by file extensions (without dots)",
					Items:       &mcp.Property{Type: "string"},
				},
				"contextLines": {
					Type:        "number",
					Description: "Number of context lines before/after matches",
					Default:     float64(2),
				},
				"maxResults": {
					Type:        "number",
					Description: "Maximum number of results to return",
					Default:     float64(100),
				},
			},
			Required: []string{"pattern", "path"},
		},
	}, handleSearchContext)

	// analyze_code tool
	server.RegisterTool(mcp.Tool{
		Name:        "analyze_code",
		Description: "Analyzes code files for complexity, dependencies, and quality metrics",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "File or directory path to analyze",
				},
				"recursive": {
					Type:        "boolean",
					Description: "Analyze subdirectories",
					Default:     true,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter by file extensions (without dots)",
					Items:       &mcp.Property{Type: "string"},
				},
			},
			Required: []string{"path"},
		},
	}, handleAnalyzeCode)

	// generate_outline tool
	server.RegisterTool(mcp.Tool{
		Name:        "generate_outline",
		Description: "Generates a code outline showing classes, functions, and imports",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "File path to generate outline for",
				},
			},
			Required: []string{"path"},
		},
	}, handleGenerateOutline)

	// cache_stats tool
	server.RegisterTool(mcp.Tool{
		Name:        "cache_stats",
		Description: "Returns cache statistics and performance metrics",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"detailed": {
					Type:        "boolean",
					Description: "Include detailed entry information",
					Default:     false,
				},
			},
		},
	}, handleCacheStats)

	// get_chunk_count tool
	server.RegisterTool(mcp.Tool{
		Name:        "get_chunk_count",
		Description: "Gets the total number of chunks for a file or directory",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "File or directory path",
				},
				"chunkSize": {
					Type:        "number",
					Description: "Size of each chunk in bytes",
					Default:     float64(DefaultChunkSize),
				},
			},
			Required: []string{"path"},
		},
	}, handleGetChunkCount)

	// getFiles tool
	server.RegisterTool(mcp.Tool{
		Name:        "getFiles",
		Description: "Batch retrieve multiple files at once",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"filePathList": {
					Type:        "array",
					Description: "List of file paths to retrieve",
					Items: &mcp.Property{
						Type: "object",
						Properties: map[string]mcp.Property{
							"fileName": {
								Type:        "string",
								Description: "Path to the file",
							},
						},
					},
				},
			},
			Required: []string{"filePathList"},
		},
	}, handleGetFiles)

	// get_folder_structure tool
	server.RegisterTool(mcp.Tool{
		Name:        "get_folder_structure",
		Description: "Returns a tree representation of the folder structure",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Directory path",
				},
				"maxDepth": {
					Type:        "number",
					Description: "Maximum depth to traverse (0 for unlimited)",
					Default:     float64(5),
				},
			},
			Required: []string{"path"},
		},
	}, handleGetFolderStructure)
}

func handleListContextFiles(args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	includeHidden := getBool(args, "includeHidden", false)
	fileTypes := getStringArray(args, "fileTypes")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	entries, err := files.ListFiles(absPath, recursive, fileTypes, includeHidden)
	if err != nil {
		return errorResult(err.Error())
	}

	result, _ := json.MarshalIndent(entries, "", "  ")
	return textResult(string(result))
}

func handleReadContext(args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)
	maxSize := getInt64(args, "maxSize", DefaultMaxSize)
	recursive := getBool(args, "recursive", true)
	fileTypes := getStringArray(args, "fileTypes")
	chunkNumber := getInt(args, "chunkNumber", 0)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Path not found: %s", err.Error()))
	}

	if info.IsDir() {
		contents, err := files.ReadDirectory(absPath, recursive, fileTypes, maxSize)
		if err != nil {
			return errorResult(err.Error())
		}
		result, _ := json.MarshalIndent(contents, "", "  ")
		return textResult(string(result))
	}

	// Check cache first
	if entry, ok := fileCache.Get(absPath); ok {
		if entry.ModifiedTime.Equal(info.ModTime()) || entry.ModifiedTime.After(info.ModTime()) {
			return textResult(entry.Content)
		}
	}

	// Handle large files with chunking
	if info.Size() > maxSize {
		content, totalChunks, err := analysis.ReadChunk(absPath, chunkNumber, DefaultChunkSize)
		if err != nil {
			return errorResult(err.Error())
		}

		result := map[string]interface{}{
			"content":      content,
			"chunkNumber":  chunkNumber,
			"totalChunks":  totalChunks,
			"path":         absPath,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return textResult(string(data))
	}

	content, err := files.ReadFile(absPath, maxSize)
	if err != nil {
		return errorResult(err.Error())
	}

	// Update cache
	fileCache.Set(absPath, &cache.Entry{
		Content:      content.Content,
		Size:         content.Metadata.Size,
		ModifiedTime: content.Metadata.ModifiedTime,
	})

	result, _ := json.MarshalIndent(content, "", "  ")
	return textResult(string(result))
}

func handleSearchContext(args map[string]interface{}) (*mcp.CallToolResult, error) {
	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	fileTypes := getStringArray(args, "fileTypes")
	contextLines := getInt(args, "contextLines", 2)
	maxResults := getInt(args, "maxResults", 100)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	results, err := files.SearchFiles(absPath, pattern, recursive, fileTypes, contextLines, maxResults)
	if err != nil {
		return errorResult(err.Error())
	}

	result, _ := json.MarshalIndent(results, "", "  ")
	return textResult(string(result))
}

func handleAnalyzeCode(args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	fileTypes := getStringArray(args, "fileTypes")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return errorResult(fmt.Sprintf("Path not found: %s", err.Error()))
	}

	if info.IsDir() {
		analyses, aggregateMetrics, err := analysis.AnalyzeDirectory(absPath, recursive, fileTypes)
		if err != nil {
			return errorResult(err.Error())
		}

		result := map[string]interface{}{
			"files":            analyses,
			"aggregateMetrics": aggregateMetrics,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return textResult(string(data))
	}

	fileAnalysis, err := analysis.AnalyzeFile(absPath)
	if err != nil {
		return errorResult(err.Error())
	}

	result, _ := json.MarshalIndent(fileAnalysis, "", "  ")
	return textResult(string(result))
}

func handleGenerateOutline(args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	outline, err := analysis.GenerateOutline(absPath)
	if err != nil {
		return errorResult(err.Error())
	}

	result, _ := json.MarshalIndent(outline, "", "  ")
	return textResult(string(result))
}

func handleCacheStats(args map[string]interface{}) (*mcp.CallToolResult, error) {
	detailed := getBool(args, "detailed", false)
	stats := fileCache.Stats(detailed)

	result, _ := json.MarshalIndent(stats, "", "  ")
	return textResult(string(result))
}

func handleGetChunkCount(args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)
	chunkSize := getInt64(args, "chunkSize", DefaultChunkSize)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	count, err := analysis.GetChunkCount(absPath, chunkSize)
	if err != nil {
		return errorResult(err.Error())
	}

	result := map[string]interface{}{
		"path":       absPath,
		"chunkCount": count,
		"chunkSize":  chunkSize,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleGetFiles(args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePathList, ok := args["filePathList"].([]interface{})
	if !ok {
		return errorResult("Invalid filePathList")
	}

	results := make(map[string]interface{})

	for _, item := range filePathList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		fileName, ok := itemMap["fileName"].(string)
		if !ok {
			continue
		}

		absPath, err := filepath.Abs(fileName)
		if err != nil {
			results[fileName] = map[string]interface{}{
				"error": fmt.Sprintf("Invalid path: %s", err.Error()),
			}
			continue
		}

		content, err := files.ReadFile(absPath, DefaultMaxSize)
		if err != nil {
			results[fileName] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		results[fileName] = content
	}

	data, _ := json.MarshalIndent(results, "", "  ")
	return textResult(string(data))
}

func handleGetFolderStructure(args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, _ := args["path"].(string)
	maxDepth := getInt(args, "maxDepth", 5)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	structure, err := analysis.GetFolderStructure(absPath, maxDepth)
	if err != nil {
		return errorResult(err.Error())
	}

	return textResult(structure)
}

// Helper functions
func textResult(text string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: text}},
	}, nil
}

func errorResult(message string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.ContentItem{{Type: "text", Text: message}},
		IsError: true,
	}, nil
}

func getBool(args map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := args[key].(bool); ok {
		return val
	}
	return defaultVal
}

func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if val, ok := args[key].(float64); ok {
		return int(val)
	}
	return defaultVal
}

func getInt64(args map[string]interface{}, key string, defaultVal int64) int64 {
	if val, ok := args[key].(float64); ok {
		return int64(val)
	}
	return defaultVal
}

func getStringArray(args map[string]interface{}, key string) []string {
	if val, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
