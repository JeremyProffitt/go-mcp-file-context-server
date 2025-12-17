package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/analysis"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/cache"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/files"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/logging"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/mcp"
)

const (
	AppName          = "go-mcp-file-context-server"
	Version          = "1.0.0"
	DefaultMaxSize   = 10 * 1024 * 1024 // 10MB
	DefaultCacheSize = 500
	DefaultCacheTTL  = 5 * time.Minute
	DefaultChunkSize = 64 * 1024 // 64KB
)

// Environment variable names
const (
	EnvLogDir   = "MCP_LOG_DIR"
	EnvLogLevel = "MCP_LOG_LEVEL"
)

var fileCache *cache.Cache
var logger *logging.Logger

func main() {
	// Parse command line flags
	logDir := flag.String("log-dir", "", "Directory for log files (default: ~/go-mcp-file-context-server/logs)")
	logLevel := flag.String("log-level", "info", "Log level: off, error, warn, info, access, debug")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("%s version %s\n", AppName, Version)
		os.Exit(0)
	}

	// Resolve log directory (CLI flag > env var > default)
	resolvedLogDir := *logDir
	if resolvedLogDir == "" {
		resolvedLogDir = os.Getenv(EnvLogDir)
	}
	if resolvedLogDir == "" {
		resolvedLogDir = logging.DefaultLogDir(AppName)
	}

	// Resolve log level (CLI flag > env var > default)
	resolvedLogLevel := *logLevel
	if envLevel := os.Getenv(EnvLogLevel); envLevel != "" && *logLevel == "info" {
		resolvedLogLevel = envLevel
	}
	parsedLogLevel := logging.ParseLogLevel(resolvedLogLevel)

	// Initialize logger
	var err error
	logger, err = logging.NewLogger(logging.Config{
		LogDir:  resolvedLogDir,
		AppName: AppName,
		Level:   parsedLogLevel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Log startup information
	startupInfo := logging.GetStartupInfo(
		Version,
		resolvedLogDir,
		parsedLogLevel,
		DefaultCacheSize,
		DefaultCacheTTL,
		DefaultMaxSize,
		DefaultChunkSize,
	)
	logger.LogStartup(startupInfo)

	// Initialize cache
	fileCache, err = cache.NewCache(DefaultCacheSize, DefaultCacheTTL)
	if err != nil {
		logger.Error("Failed to initialize cache: %v", err)
		fmt.Fprintf(os.Stderr, "Failed to initialize cache: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Cache initialized: size=%d, ttl=%s", DefaultCacheSize, DefaultCacheTTL)

	// Create MCP server
	server := mcp.NewServer("file-context-server", Version)
	logger.Info("MCP server created: name=%s, version=%s", "file-context-server", Version)

	// Register tools
	registerTools(server)
	logger.Info("Tools registered successfully")

	// Run the server
	logger.Info("Starting MCP server...")
	if err := server.Run(); err != nil {
		logger.Error("Server error: %v", err)
		logger.LogShutdown(fmt.Sprintf("error: %v", err))
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}

	logger.LogShutdown("normal exit")
}

func printHelp() {
	fmt.Printf(`%s - MCP File Context Server

A Model Context Protocol (MCP) server that provides file system context to LLMs.

USAGE:
    %s [OPTIONS]

OPTIONS:
    -log-dir <path>     Directory for log files
                        Default: ~/go-mcp-file-context-server/logs
                        Env: MCP_LOG_DIR

    -log-level <level>  Log level: off, error, warn, info, access, debug
                        Default: info
                        Env: MCP_LOG_LEVEL

    -version            Show version information

    -help               Show this help message

ENVIRONMENT VARIABLES:
    MCP_LOG_DIR         Override default log directory
    MCP_LOG_LEVEL       Override default log level

LOG LEVELS:
    off      Disable all logging
    error    Log errors only
    warn     Log warnings and errors
    info     Log general information (default)
    access   Log file access operations (includes bytes read/written)
    debug    Log detailed debugging information

EXAMPLES:
    # Run with default settings
    %s

    # Run with custom log directory
    %s -log-dir /var/log/mcp

    # Run with debug logging
    %s -log-level debug

    # Using environment variables
    MCP_LOG_DIR=/var/log/mcp MCP_LOG_LEVEL=access %s

`, AppName, AppName, AppName, AppName, AppName, AppName)
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
	logger.ToolCall("list_context_files", args)

	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	includeHidden := getBool(args, "includeHidden", false)
	fileTypes := getStringArray(args, "fileTypes")

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("list_context_files: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	entries, err := files.ListFiles(absPath, recursive, fileTypes, includeHidden)
	if err != nil {
		logger.Error("list_context_files: failed to list files in %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.DirectoryRead(absPath, len(entries), nil)
	logger.Debug("list_context_files: listed %d files from %q", len(entries), absPath)

	result, _ := json.MarshalIndent(entries, "", "  ")
	return textResult(string(result))
}

func handleReadContext(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("read_context", args)

	path, _ := args["path"].(string)
	maxSize := getInt64(args, "maxSize", DefaultMaxSize)
	recursive := getBool(args, "recursive", true)
	fileTypes := getStringArray(args, "fileTypes")
	chunkNumber := getInt(args, "chunkNumber", 0)

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("read_context: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		logger.Error("read_context: path not found %q: %v", absPath, err)
		return errorResult(fmt.Sprintf("Path not found: %s", err.Error()))
	}

	if info.IsDir() {
		contents, err := files.ReadDirectory(absPath, recursive, fileTypes, maxSize)
		if err != nil {
			logger.Error("read_context: failed to read directory %q: %v", absPath, err)
			return errorResult(err.Error())
		}
		logger.DirectoryRead(absPath, len(contents), nil)
		result, _ := json.MarshalIndent(contents, "", "  ")
		return textResult(string(result))
	}

	// Check cache first
	if entry, ok := fileCache.Get(absPath); ok {
		if entry.ModifiedTime.Equal(info.ModTime()) || entry.ModifiedTime.After(info.ModTime()) {
			logger.CacheHit(absPath)
			logger.FileRead(absPath, entry.Size, nil)
			return textResult(entry.Content)
		}
	}
	logger.CacheMiss(absPath)

	// Handle large files with chunking
	if info.Size() > maxSize {
		content, totalChunks, err := analysis.ReadChunk(absPath, chunkNumber, DefaultChunkSize)
		if err != nil {
			logger.Error("read_context: failed to read chunk %d of %q: %v", chunkNumber, absPath, err)
			return errorResult(err.Error())
		}

		bytesRead := int64(len(content))
		logger.FileRead(absPath, bytesRead, nil)
		logger.Debug("read_context: read chunk %d/%d from %q (%d bytes)", chunkNumber+1, totalChunks, absPath, bytesRead)

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
		logger.Error("read_context: failed to read file %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.FileRead(absPath, content.Metadata.Size, nil)
	logger.Debug("read_context: read file %q (%d bytes)", absPath, content.Metadata.Size)

	// Update cache
	fileCache.Set(absPath, &cache.Entry{
		Content:      content.Content,
		Size:         content.Metadata.Size,
		ModifiedTime: content.Metadata.ModifiedTime,
	})
	logger.CacheSet(absPath, content.Metadata.Size)

	result, _ := json.MarshalIndent(content, "", "  ")
	return textResult(string(result))
}

func handleSearchContext(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("search_context", args)

	pattern, _ := args["pattern"].(string)
	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	fileTypes := getStringArray(args, "fileTypes")
	contextLines := getInt(args, "contextLines", 2)
	maxResults := getInt(args, "maxResults", 100)

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("search_context: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	results, err := files.SearchFiles(absPath, pattern, recursive, fileTypes, contextLines, maxResults)
	if err != nil {
		logger.Error("search_context: failed to search in %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.Search(absPath, pattern, results.Total, nil)
	logger.Debug("search_context: found %d matches for pattern %q in %q", results.Total, pattern, absPath)

	result, _ := json.MarshalIndent(results, "", "  ")
	return textResult(string(result))
}

func handleAnalyzeCode(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("analyze_code", args)

	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	fileTypes := getStringArray(args, "fileTypes")

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("analyze_code: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	info, err := os.Stat(absPath)
	if err != nil {
		logger.Error("analyze_code: path not found %q: %v", absPath, err)
		return errorResult(fmt.Sprintf("Path not found: %s", err.Error()))
	}

	if info.IsDir() {
		analyses, aggregateMetrics, err := analysis.AnalyzeDirectory(absPath, recursive, fileTypes)
		if err != nil {
			logger.Error("analyze_code: failed to analyze directory %q: %v", absPath, err)
			return errorResult(err.Error())
		}

		logger.DirectoryRead(absPath, len(analyses), nil)
		logger.Debug("analyze_code: analyzed %d files in %q", len(analyses), absPath)

		result := map[string]interface{}{
			"files":            analyses,
			"aggregateMetrics": aggregateMetrics,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return textResult(string(data))
	}

	fileAnalysis, err := analysis.AnalyzeFile(absPath)
	if err != nil {
		logger.Error("analyze_code: failed to analyze file %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.FileRead(absPath, info.Size(), nil)
	logger.Debug("analyze_code: analyzed file %q", absPath)

	result, _ := json.MarshalIndent(fileAnalysis, "", "  ")
	return textResult(string(result))
}

func handleGenerateOutline(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("generate_outline", args)

	path, _ := args["path"].(string)

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("generate_outline: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	outline, err := analysis.GenerateOutline(absPath)
	if err != nil {
		logger.Error("generate_outline: failed to generate outline for %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.Debug("generate_outline: generated outline for %q", absPath)

	result, _ := json.MarshalIndent(outline, "", "  ")
	return textResult(string(result))
}

func handleCacheStats(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("cache_stats", args)

	detailed := getBool(args, "detailed", false)
	stats := fileCache.Stats(detailed)

	logger.Debug("cache_stats: retrieved cache statistics (detailed=%v)", detailed)

	result, _ := json.MarshalIndent(stats, "", "  ")
	return textResult(string(result))
}

func handleGetChunkCount(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("get_chunk_count", args)

	path, _ := args["path"].(string)
	chunkSize := getInt64(args, "chunkSize", DefaultChunkSize)

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("get_chunk_count: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	count, err := analysis.GetChunkCount(absPath, chunkSize)
	if err != nil {
		logger.Error("get_chunk_count: failed to get chunk count for %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.Debug("get_chunk_count: %q has %d chunks (chunkSize=%d)", absPath, count, chunkSize)

	result := map[string]interface{}{
		"path":       absPath,
		"chunkCount": count,
		"chunkSize":  chunkSize,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleGetFiles(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("getFiles", args)

	filePathList, ok := args["filePathList"].([]interface{})
	if !ok {
		logger.Error("getFiles: invalid filePathList parameter")
		return errorResult("Invalid filePathList")
	}

	logger.Debug("getFiles: processing %d files", len(filePathList))
	results := make(map[string]interface{})
	var totalBytesRead int64

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
			logger.Error("getFiles: invalid path %q: %v", fileName, err)
			results[fileName] = map[string]interface{}{
				"error": fmt.Sprintf("Invalid path: %s", err.Error()),
			}
			continue
		}

		content, err := files.ReadFile(absPath, DefaultMaxSize)
		if err != nil {
			logger.Error("getFiles: failed to read file %q: %v", absPath, err)
			logger.FileRead(absPath, 0, err)
			results[fileName] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		logger.FileRead(absPath, content.Metadata.Size, nil)
		totalBytesRead += content.Metadata.Size
		results[fileName] = content
	}

	logger.Debug("getFiles: read %d files, total %d bytes", len(results), totalBytesRead)

	data, _ := json.MarshalIndent(results, "", "  ")
	return textResult(string(data))
}

func handleGetFolderStructure(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("get_folder_structure", args)

	path, _ := args["path"].(string)
	maxDepth := getInt(args, "maxDepth", 5)

	absPath, err := filepath.Abs(path)
	if err != nil {
		logger.Error("get_folder_structure: invalid path %q: %v", path, err)
		return errorResult(fmt.Sprintf("Invalid path: %s", err.Error()))
	}

	structure, err := analysis.GetFolderStructure(absPath, maxDepth)
	if err != nil {
		logger.Error("get_folder_structure: failed to get structure for %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.Debug("get_folder_structure: generated structure for %q (maxDepth=%d)", absPath, maxDepth)

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
