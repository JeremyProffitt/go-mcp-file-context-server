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
	EnvRootDir  = "MCP_ROOT_DIR"
)

var fileCache *cache.Cache
var logger *logging.Logger
var allowedRootDir string // If set, restricts all file operations to this directory

func main() {
	// Parse command line flags
	logDir := flag.String("log-dir", "", "Directory for log files (default: ~/go-mcp-file-context-server/logs)")
	logLevel := flag.String("log-level", "info", "Log level: off, error, warn, info, access, debug")
	rootDir := flag.String("root-dir", "", "Root directory to restrict file access (default: no restriction)")
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

	// Resolve log directory (CLI flag > env var > default) and track source
	var resolvedLogDir string
	var logDirSource logging.ConfigSource
	if *logDir != "" {
		resolvedLogDir = *logDir
		logDirSource = logging.SourceFlag
	} else if envVal := os.Getenv(EnvLogDir); envVal != "" {
		resolvedLogDir = envVal
		logDirSource = logging.SourceEnvironment
	} else {
		resolvedLogDir = logging.DefaultLogDir(AppName)
		logDirSource = logging.SourceDefault
	}

	// Resolve log level (CLI flag > env var > default) and track source
	var resolvedLogLevel string
	var logLevelSource logging.ConfigSource
	if *logLevel != "info" {
		// Non-default flag value means flag was explicitly set
		resolvedLogLevel = *logLevel
		logLevelSource = logging.SourceFlag
	} else if envVal := os.Getenv(EnvLogLevel); envVal != "" {
		resolvedLogLevel = envVal
		logLevelSource = logging.SourceEnvironment
	} else {
		resolvedLogLevel = "info"
		logLevelSource = logging.SourceDefault
	}
	parsedLogLevel := logging.ParseLogLevel(resolvedLogLevel)

	// Resolve root directory (CLI flag > env var > no restriction) and track source
	var resolvedRootDir string
	var rootDirSource logging.ConfigSource
	if *rootDir != "" {
		resolvedRootDir = *rootDir
		rootDirSource = logging.SourceFlag
	} else if envVal := os.Getenv(EnvRootDir); envVal != "" {
		resolvedRootDir = envVal
		rootDirSource = logging.SourceEnvironment
	} else {
		resolvedRootDir = ""
		rootDirSource = logging.SourceDefault
	}
	if resolvedRootDir != "" {
		absRoot, err := filepath.Abs(resolvedRootDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid root directory %q: %v\n", resolvedRootDir, err)
			os.Exit(1)
		}
		// Verify the directory exists
		info, err := os.Stat(absRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Root directory does not exist %q: %v\n", absRoot, err)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Root path is not a directory: %q\n", absRoot)
			os.Exit(1)
		}
		allowedRootDir = absRoot
	}

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

	// Log startup information with configuration sources
	startupInfo := logging.GetStartupInfo(
		Version,
		logging.ConfigValue{Value: resolvedLogDir, Source: logDirSource},
		logging.ConfigValue{Value: resolvedLogLevel, Source: logLevelSource},
		logging.ConfigValue{Value: allowedRootDir, Source: rootDirSource},
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

	// Log root directory restriction
	if allowedRootDir != "" {
		logger.Info("Root directory restriction enabled: %s", allowedRootDir)
	} else {
		logger.Info("Root directory restriction: disabled (full filesystem access)")
	}

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
    -root-dir <path>    Root directory to restrict file access
                        Default: no restriction (full filesystem access)
                        Env: MCP_ROOT_DIR

    -log-dir <path>     Directory for log files
                        Default: ~/go-mcp-file-context-server/logs
                        Env: MCP_LOG_DIR

    -log-level <level>  Log level: off, error, warn, info, access, debug
                        Default: info
                        Env: MCP_LOG_LEVEL

    -version            Show version information

    -help               Show this help message

ENVIRONMENT VARIABLES:
    MCP_ROOT_DIR        Restrict file access to this directory
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
    # Run with default settings (full filesystem access)
    %s

    # Restrict access to a specific project directory
    %s -root-dir /home/user/myproject

    # Run with custom log directory
    %s -log-dir /var/log/mcp

    # Run with debug logging
    %s -log-level debug

    # Using environment variables
    MCP_ROOT_DIR=/home/user/project MCP_LOG_LEVEL=access %s

`, AppName, AppName, AppName, AppName, AppName, AppName, AppName)
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

	// write_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "write_file",
		Description: "Create a new file or overwrite an existing file with new content. Creates parent directories if they don't exist.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file to write",
				},
				"content": {
					Type:        "string",
					Description: "Content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
	}, handleWriteFile)

	// create_directory tool
	server.RegisterTool(mcp.Tool{
		Name:        "create_directory",
		Description: "Create a new directory or ensure a directory exists. Creates parent directories if needed.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Path to the directory to create",
				},
			},
			Required: []string{"path"},
		},
	}, handleCreateDirectory)

	// copy_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "copy_file",
		Description: "Copy a file or directory from source to destination. Supports recursive copying for directories.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"source": {
					Type:        "string",
					Description: "Source file or directory path",
				},
				"destination": {
					Type:        "string",
					Description: "Destination path",
				},
			},
			Required: []string{"source", "destination"},
		},
	}, handleCopyFile)

	// move_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "move_file",
		Description: "Move or rename a file or directory from source to destination.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"source": {
					Type:        "string",
					Description: "Source file or directory path",
				},
				"destination": {
					Type:        "string",
					Description: "Destination path",
				},
			},
			Required: []string{"source", "destination"},
		},
	}, handleMoveFile)

	// delete_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file or directory from the file system.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file or directory to delete",
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true, delete directories recursively (required for non-empty directories)",
					Default:     false,
				},
			},
			Required: []string{"path"},
		},
	}, handleDeleteFile)

	// modify_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "modify_file",
		Description: "Find and replace text in a file. Supports both literal string matching and regular expressions.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Path to the file to modify",
				},
				"find": {
					Type:        "string",
					Description: "Text or regex pattern to find",
				},
				"replace": {
					Type:        "string",
					Description: "Replacement text",
				},
				"all_occurrences": {
					Type:        "boolean",
					Description: "If true, replace all occurrences; otherwise replace only the first",
					Default:     true,
				},
				"regex": {
					Type:        "boolean",
					Description: "If true, treat 'find' as a regular expression",
					Default:     false,
				},
			},
			Required: []string{"path", "find", "replace"},
		},
	}, handleModifyFile)
}

func handleListContextFiles(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("list_context_files", args)

	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", true)
	includeHidden := getBool(args, "includeHidden", false)
	fileTypes := getStringArray(args, "fileTypes")

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("list_context_files: %v", err)
		return errorResult(err.Error())
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

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("read_context: %v", err)
		return errorResult(err.Error())
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

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("search_context: %v", err)
		return errorResult(err.Error())
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

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("analyze_code: %v", err)
		return errorResult(err.Error())
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

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("generate_outline: %v", err)
		return errorResult(err.Error())
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

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("get_chunk_count: %v", err)
		return errorResult(err.Error())
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

		absPath, err := validatePath(fileName)
		if err != nil {
			logger.Error("getFiles: %v", err)
			results[fileName] = map[string]interface{}{
				"error": err.Error(),
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

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("get_folder_structure: %v", err)
		return errorResult(err.Error())
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

// validatePath checks if the given path is within the allowed root directory.
// Returns the absolute path if valid, or an error if access is denied.
func validatePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// If no root directory restriction, allow all paths
	if allowedRootDir == "" {
		return absPath, nil
	}

	// Check if the absolute path is within the allowed root directory
	// Use filepath.Rel to check if path is relative to root (not starting with ..)
	relPath, err := filepath.Rel(allowedRootDir, absPath)
	if err != nil {
		return "", fmt.Errorf("access denied: path outside allowed directory")
	}

	// If the relative path starts with "..", it's outside the root
	if len(relPath) >= 2 && relPath[:2] == ".." {
		return "", fmt.Errorf("access denied: path %q is outside allowed directory %q", path, allowedRootDir)
	}

	// Also check for absolute paths that might have been crafted to escape
	if !isSubPath(allowedRootDir, absPath) {
		return "", fmt.Errorf("access denied: path %q is outside allowed directory %q", path, allowedRootDir)
	}

	return absPath, nil
}

// isSubPath checks if child is a subpath of parent
func isSubPath(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)

	// Ensure parent ends with separator for proper prefix matching
	if parent[len(parent)-1] != filepath.Separator {
		parent = parent + string(filepath.Separator)
	}

	// Child is a subpath if it starts with parent or equals parent (without trailing separator)
	return child == filepath.Clean(parent[:len(parent)-1]) || len(child) >= len(parent) && child[:len(parent)] == parent
}

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

func getString(args map[string]interface{}, key string, defaultVal string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return defaultVal
}

func handleWriteFile(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("write_file", args)

	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("write_file: %v", err)
		return errorResult(err.Error())
	}

	result, err := files.WriteFile(absPath, content)
	if err != nil {
		logger.Error("write_file: failed to write file %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	action := "overwrote"
	if result.Created {
		action = "created"
	}
	logger.Info("write_file: %s file %q (%d bytes)", action, absPath, result.BytesWritten)

	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleCreateDirectory(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("create_directory", args)

	path, _ := args["path"].(string)

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("create_directory: %v", err)
		return errorResult(err.Error())
	}

	if err := files.CreateDirectory(absPath); err != nil {
		logger.Error("create_directory: failed to create directory %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	logger.Info("create_directory: created directory %q", absPath)

	result := map[string]interface{}{
		"path":    absPath,
		"created": true,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleCopyFile(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("copy_file", args)

	source, _ := args["source"].(string)
	destination, _ := args["destination"].(string)

	absSrc, err := validatePath(source)
	if err != nil {
		logger.Error("copy_file: source %v", err)
		return errorResult(err.Error())
	}

	absDst, err := validatePath(destination)
	if err != nil {
		logger.Error("copy_file: destination %v", err)
		return errorResult(err.Error())
	}

	result, err := files.CopyFile(absSrc, absDst)
	if err != nil {
		logger.Error("copy_file: failed to copy %q to %q: %v", absSrc, absDst, err)
		return errorResult(err.Error())
	}

	logger.Info("copy_file: copied %q to %q (%d bytes)", absSrc, absDst, result.BytesCopied)

	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleMoveFile(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("move_file", args)

	source, _ := args["source"].(string)
	destination, _ := args["destination"].(string)

	absSrc, err := validatePath(source)
	if err != nil {
		logger.Error("move_file: source %v", err)
		return errorResult(err.Error())
	}

	absDst, err := validatePath(destination)
	if err != nil {
		logger.Error("move_file: destination %v", err)
		return errorResult(err.Error())
	}

	result, err := files.MoveFile(absSrc, absDst)
	if err != nil {
		logger.Error("move_file: failed to move %q to %q: %v", absSrc, absDst, err)
		return errorResult(err.Error())
	}

	logger.Info("move_file: moved %q to %q", absSrc, absDst)

	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleDeleteFile(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("delete_file", args)

	path, _ := args["path"].(string)
	recursive := getBool(args, "recursive", false)

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("delete_file: %v", err)
		return errorResult(err.Error())
	}

	result, err := files.DeleteFile(absPath, recursive)
	if err != nil {
		logger.Error("delete_file: failed to delete %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	itemType := "file"
	if result.IsDirectory {
		itemType = "directory"
	}
	logger.Info("delete_file: deleted %s %q", itemType, absPath)

	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}

func handleModifyFile(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("modify_file", args)

	path, _ := args["path"].(string)
	find, _ := args["find"].(string)
	replace := getString(args, "replace", "")
	allOccurrences := getBool(args, "all_occurrences", true)
	useRegex := getBool(args, "regex", false)

	absPath, err := validatePath(path)
	if err != nil {
		logger.Error("modify_file: %v", err)
		return errorResult(err.Error())
	}

	result, err := files.ModifyFile(absPath, find, replace, allOccurrences, useRegex)
	if err != nil {
		logger.Error("modify_file: failed to modify %q: %v", absPath, err)
		return errorResult(err.Error())
	}

	if result.Modified {
		logger.Info("modify_file: modified %q (%d replacements)", absPath, result.Replacements)
	} else {
		logger.Info("modify_file: no changes made to %q", absPath)
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return textResult(string(data))
}
