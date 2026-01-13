package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/analysis"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/cache"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/files"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/logging"
	"github.com/JeremyProffitt/go-mcp-file-context-server/pkg/mcp"
	"github.com/bmatcuk/doublestar/v4"
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
	EnvLogDir          = "MCP_LOG_DIR"
	EnvLogLevel        = "MCP_LOG_LEVEL"
	EnvRootDir         = "MCP_ROOT_DIR"
	EnvBlockedPatterns = "MCP_BLOCKED_PATTERNS"
)

// DefaultBlockedPatterns are blocked by default for security
var DefaultBlockedPatterns = []string{
	".aws/*",
	".env",
	".mcp_env",
}

var fileCache *cache.Cache
var logger *logging.Logger
var allowedRootDirs []string // If set, restricts all file operations to these directories
var blockedPatterns []string // Patterns to block access to

func main() {
	// Load environment variables from ~/.mcp_env if it exists
	// This must happen before flag parsing so env vars are available for defaults
	logging.LoadEnvFile()

	// Parse command line flags
	logDir := flag.String("log-dir", "", "Directory for log files (default: ~/go-mcp-file-context-server/logs)")
	logLevel := flag.String("log-level", "info", "Log level: off, error, warn, info, access, debug")
	rootDir := flag.String("root-dir", "", "Root directories to restrict file access, comma-separated (default: no restriction)")
	blockedPatternsFlag := flag.String("blocked-patterns", "", "Patterns to block, comma-separated (default: .aws/*,.env,.mcp_env)")
	httpMode := flag.Bool("http", false, "Run in HTTP mode instead of stdio")
	httpPort := flag.Int("port", 3000, "HTTP port (only used with --http)")
	httpHost := flag.String("host", "127.0.0.1", "HTTP host (only used with --http)")
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
	var addAppSubfolder bool
	if *logDir != "" {
		resolvedLogDir = *logDir
		logDirSource = logging.SourceFlag
		addAppSubfolder = true // User specified a shared log directory
	} else if envVal := os.Getenv(EnvLogDir); envVal != "" {
		resolvedLogDir = envVal
		logDirSource = logging.SourceEnvironment
		addAppSubfolder = true // User specified a shared log directory
	} else {
		resolvedLogDir = logging.DefaultLogDir(AppName)
		logDirSource = logging.SourceDefault
		addAppSubfolder = false // Default already includes app name
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

	// Resolve root directories (CLI flag > env var > no restriction) and track source
	var resolvedRootDirs string
	var rootDirSource logging.ConfigSource
	if *rootDir != "" {
		resolvedRootDirs = *rootDir
		rootDirSource = logging.SourceFlag
	} else if envVal := os.Getenv(EnvRootDir); envVal != "" {
		resolvedRootDirs = envVal
		rootDirSource = logging.SourceEnvironment
	} else {
		resolvedRootDirs = ""
		rootDirSource = logging.SourceDefault
	}

	// Parse comma-separated root directories
	if resolvedRootDirs != "" {
		for _, dir := range parseCommaSeparated(resolvedRootDirs) {
			expanded := logging.ExpandPath(dir)
			absRoot, err := filepath.Abs(expanded)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid root directory %q: %v\n", dir, err)
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
			allowedRootDirs = append(allowedRootDirs, absRoot)
		}
	}

	// Resolve blocked patterns (CLI flag > env var > defaults)
	var resolvedBlockedPatterns string
	var blockedPatternsSource logging.ConfigSource
	var blockedPatternsSet bool
	if *blockedPatternsFlag != "" {
		resolvedBlockedPatterns = *blockedPatternsFlag
		blockedPatternsSource = logging.SourceFlag
		blockedPatternsSet = true
	} else if envVal, exists := os.LookupEnv(EnvBlockedPatterns); exists {
		resolvedBlockedPatterns = envVal
		blockedPatternsSource = logging.SourceEnvironment
		blockedPatternsSet = true
	} else {
		blockedPatternsSource = logging.SourceDefault
		blockedPatternsSet = false
	}

	// Parse blocked patterns - use defaults if not explicitly set
	if blockedPatternsSet {
		blockedPatterns = parseCommaSeparated(resolvedBlockedPatterns)
	} else {
		blockedPatterns = DefaultBlockedPatterns
	}

	// Initialize logger
	var err error
	logger, err = logging.NewLogger(logging.Config{
		LogDir:          resolvedLogDir,
		AppName:         AppName,
		Level:           parsedLogLevel,
		AddAppSubfolder: addAppSubfolder,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Log startup information with configuration sources
	rootDirsStr := strings.Join(allowedRootDirs, ", ")
	if rootDirsStr == "" {
		rootDirsStr = "(no restriction)"
	}
	blockedPatternsStr := strings.Join(blockedPatterns, ", ")
	if blockedPatternsStr == "" {
		blockedPatternsStr = "(none)"
	}
	startupInfo := logging.GetStartupInfo(
		Version,
		logging.ConfigValue{Value: resolvedLogDir, Source: logDirSource},
		logging.ConfigValue{Value: resolvedLogLevel, Source: logLevelSource},
		logging.ConfigValue{Value: rootDirsStr, Source: rootDirSource},
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
	if len(allowedRootDirs) > 0 {
		logger.Info("Root directory restriction enabled: %s", rootDirsStr)
	} else {
		logger.Info("Root directory restriction: disabled (full filesystem access)")
	}

	// Log blocked patterns
	logger.Info("Blocked patterns (%s): %s", blockedPatternsSource, blockedPatternsStr)

	// Create MCP server
	server := mcp.NewServer("file-context-server", Version)
	logger.Info("MCP server created: name=%s, version=%s", "file-context-server", Version)

	// Register tools
	registerTools(server)
	logger.Info("Tools registered successfully")

	// Run the server
	logger.Info("Starting MCP server...")
	if *httpMode {
		addr := fmt.Sprintf("%s:%d", *httpHost, *httpPort)
		logger.Info("Starting HTTP server on %s", addr)
		if err := server.RunHTTP(addr); err != nil {
			logger.Error("HTTP server error: %v", err)
			logger.LogShutdown(fmt.Sprintf("error: %v", err))
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := server.Run(); err != nil {
			logger.Error("Server error: %v", err)
			logger.LogShutdown(fmt.Sprintf("error: %v", err))
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}

	logger.LogShutdown("normal exit")
}

func printHelp() {
	fmt.Printf(`%s - MCP File Context Server

A Model Context Protocol (MCP) server that provides file system context to LLMs.

USAGE:
    %s [OPTIONS]

OPTIONS:
    -root-dir <paths>   Root directories to restrict file access (comma-separated)
                        Default: no restriction (full filesystem access)
                        Env: MCP_ROOT_DIR

    -blocked-patterns <patterns>
                        Patterns to block access to (comma-separated globs)
                        Default: .aws/*,.env,.mcp_env
                        Env: MCP_BLOCKED_PATTERNS

    -log-dir <path>     Directory for log files
                        Default: ~/go-mcp-file-context-server/logs
                        Env: MCP_LOG_DIR

    -log-level <level>  Log level: off, error, warn, info, access, debug
                        Default: info
                        Env: MCP_LOG_LEVEL

    -version            Show version information

    -help               Show this help message

ENVIRONMENT VARIABLES:
    MCP_ROOT_DIR           Restrict file access to these directories (comma-separated)
    MCP_BLOCKED_PATTERNS   Block access to files matching these patterns (comma-separated)
                           Default: .aws/*,.env,.mcp_env
                           Set to empty string to disable blocking
    MCP_LOG_DIR            Override default log directory
    MCP_LOG_LEVEL          Override default log level

LOG LEVELS:
    off      Disable all logging
    error    Log errors only
    warn     Log warnings and errors
    info     Log general information (default)
    access   Log file access operations (includes bytes read/written)
    debug    Log detailed debugging information

EXAMPLES:
    # Run with default settings (full filesystem access, default blocked patterns)
    %s

    # Restrict access to a specific project directory
    %s -root-dir /home/user/myproject

    # Restrict access to multiple directories
    %s -root-dir "/home/user/project1,/home/user/project2,~/shared"

    # Custom blocked patterns (replaces defaults)
    %s -blocked-patterns ".env,secrets/*,*.pem,*.key"

    # Disable all blocked patterns
    MCP_BLOCKED_PATTERNS="" %s

    # Run with custom log directory
    %s -log-dir /var/log/mcp

    # Run with debug logging
    %s -log-level debug

    # Using environment variables
    MCP_ROOT_DIR=~/projects,~/work MCP_LOG_LEVEL=access %s

`, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName, AppName)
}

func registerTools(server *mcp.Server) {
	// list_allowed_directories tool - returns configured access restrictions
	server.RegisterTool(mcp.Tool{
		Name:        "list_allowed_directories",
		Description: "Returns the list of allowed root directories and blocked patterns configured for this server. Use this tool first to understand what paths are accessible before attempting file operations.",
		InputSchema: mcp.JSONSchema{
			Type:       "object",
			Properties: map[string]mcp.Property{},
		},
		Annotations: readOnlyAnnotations(),
	}, handleListAllowedDirectories)

	// list_context_files tool
	server.RegisterTool(mcp.Tool{
		Name:        "list_context_files",
		Description: "Lists files in a directory with detailed metadata (name, size, modification time, type). Use this when you need to discover what files exist in a directory and their properties. For reading actual file contents, use read_context instead. For a visual tree representation, use get_folder_structure.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the directory to list files from",
					Examples:    []interface{}{"/home/user/project", "./src", "C:\\Users\\project"},
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true, includes files from all subdirectories recursively. If false, only lists files in the immediate directory.",
					Default:     true,
				},
				"includeHidden": {
					Type:        "boolean",
					Description: "Include hidden files (files starting with '.')",
					Default:     false,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter results to only include files with these extensions (without leading dots)",
					Items:       &mcp.Property{Type: "string"},
					Examples:    []interface{}{[]string{"go", "ts", "py"}, []string{"json", "yaml", "yml"}},
				},
			},
			Required: []string{"path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleListContextFiles)

	// read_context tool
	server.RegisterTool(mcp.Tool{
		Name:        "read_context",
		Description: "Reads and returns the actual contents of a file or directory. For a single file: returns the file content with metadata. For a directory: returns contents of all matching files. Large files are automatically chunked - use chunkNumber to paginate. Results are cached for performance. Use this when you need to examine actual file contents, not just metadata.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file or directory to read",
					Examples:    []interface{}{"/home/user/project/main.go", "./src/index.ts", "C:\\Users\\project\\README.md"},
				},
				"maxSize": {
					Type:        "integer",
					Description: "Maximum file size in bytes. Files larger than this will be chunked.",
					Default:     float64(DefaultMaxSize),
					Minimum:     int64Ptr(1),
					Maximum:     int64Ptr(100 * 1024 * 1024), // 100MB
				},
				"encoding": {
					Type:        "string",
					Description: "Character encoding for reading text files",
					Default:     "utf8",
					Examples:    []interface{}{"utf8", "ascii", "latin1"},
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true and path is a directory, includes files from all subdirectories recursively. If false, only reads files in the immediate directory.",
					Default:     true,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter results to only include files with these extensions (without leading dots)",
					Items:       &mcp.Property{Type: "string"},
					Examples:    []interface{}{[]string{"go", "ts", "py"}, []string{"json", "yaml", "yml"}},
				},
				"chunkNumber": {
					Type:        "integer",
					Description: "For large files that exceed maxSize, specify which chunk to retrieve (0-indexed). Use get_chunk_count to determine total chunks.",
					Default:     float64(0),
					Minimum:     int64Ptr(0),
				},
			},
			Required: []string{"path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleReadContext)

	// search_context tool
	server.RegisterTool(mcp.Tool{
		Name:        "search_context",
		Description: "Searches for regex patterns in file contents and returns matching lines with surrounding context. Use this to find specific code patterns, function definitions, variable usages, or any text pattern across multiple files.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"pattern": {
					Type:        "string",
					Description: "Regular expression pattern to search for. Supports standard regex syntax.",
					Examples:    []interface{}{"func\\s+\\w+", "TODO|FIXME", "import.*react"},
				},
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the directory to search in",
					Examples:    []interface{}{"/home/user/project", "./src", "C:\\Users\\project"},
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true, searches in all subdirectories recursively. If false, only searches files in the immediate directory.",
					Default:     true,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter results to only search files with these extensions (without leading dots)",
					Items:       &mcp.Property{Type: "string"},
					Examples:    []interface{}{[]string{"go", "ts", "py"}, []string{"json", "yaml", "yml"}},
				},
				"contextLines": {
					Type:        "integer",
					Description: "Number of lines to include before and after each match for context",
					Default:     float64(2),
					Minimum:     int64Ptr(0),
					Maximum:     int64Ptr(50),
				},
				"maxResults": {
					Type:        "integer",
					Description: "Maximum number of matching results to return. Use to limit output for common patterns.",
					Default:     float64(100),
					Minimum:     int64Ptr(1),
					Maximum:     int64Ptr(10000),
				},
			},
			Required: []string{"pattern", "path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleSearchContext)

	// analyze_code tool
	server.RegisterTool(mcp.Tool{
		Name:        "analyze_code",
		Description: "Analyzes code files and returns metrics including complexity scores, dependency counts, lines of code, and quality indicators. For directories, provides aggregate metrics across all files. Use this to assess code quality and identify complex areas.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file or directory to analyze",
					Examples:    []interface{}{"/home/user/project/main.go", "./src", "C:\\Users\\project"},
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true and path is a directory, analyzes files in all subdirectories recursively. If false, only analyzes files in the immediate directory.",
					Default:     true,
				},
				"fileTypes": {
					Type:        "array",
					Description: "Filter results to only analyze files with these extensions (without leading dots)",
					Items:       &mcp.Property{Type: "string"},
					Examples:    []interface{}{[]string{"go", "ts", "py"}, []string{"js", "jsx", "tsx"}},
				},
			},
			Required: []string{"path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleAnalyzeCode)

	// generate_outline tool
	server.RegisterTool(mcp.Tool{
		Name:        "generate_outline",
		Description: "Generates a structural outline of a code file showing imports, classes, functions, and methods with their line numbers. Use this to quickly understand the structure of a file without reading its full contents.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to generate an outline for",
					Examples:    []interface{}{"/home/user/project/main.go", "./src/index.ts", "C:\\Users\\project\\app.py"},
				},
			},
			Required: []string{"path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleGenerateOutline)

	// cache_stats tool
	server.RegisterTool(mcp.Tool{
		Name:        "cache_stats",
		Description: "Returns cache statistics including hit/miss rates, memory usage, and optionally details about cached entries. Use this to monitor cache performance and diagnose caching issues.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"detailed": {
					Type:        "boolean",
					Description: "If true, includes a list of all cached file entries with their sizes and ages. If false, returns only aggregate statistics.",
					Default:     false,
				},
			},
		},
		Annotations: readOnlyAnnotations(),
	}, handleCacheStats)

	// get_chunk_count tool
	server.RegisterTool(mcp.Tool{
		Name:        "get_chunk_count",
		Description: "Returns the total number of chunks needed to read a large file. Use this before calling read_context with chunkNumber to know how many iterations are needed to read the complete file.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to get chunk count for",
					Examples:    []interface{}{"/home/user/project/large-file.log", "./data/export.csv"},
				},
				"chunkSize": {
					Type:        "integer",
					Description: "Size of each chunk in bytes. Must match the chunkSize used in read_context for consistent pagination.",
					Default:     float64(DefaultChunkSize),
					Minimum:     int64Ptr(1024),          // 1KB minimum
					Maximum:     int64Ptr(10 * 1024 * 1024), // 10MB maximum
				},
			},
			Required: []string{"path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleGetChunkCount)

	// get_files tool (batch file retrieval)
	server.RegisterTool(mcp.Tool{
		Name:        "get_files",
		Description: "Batch retrieve contents of multiple files in a single request. More efficient than calling read_context multiple times when you need to read several known files. Returns a map of file paths to their contents.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"filePathList": {
					Type:        "array",
					Description: "List of file path objects to retrieve",
					Items: &mcp.Property{
						Type: "object",
						Properties: map[string]mcp.Property{
							"fileName": {
								Type:        "string",
								Description: "Absolute or relative path to the file to retrieve",
								Examples:    []interface{}{"/home/user/project/main.go", "./src/index.ts"},
							},
						},
					},
					Examples: []interface{}{
						[]map[string]string{{"fileName": "./src/main.go"}, {"fileName": "./src/utils.go"}},
					},
				},
			},
			Required: []string{"filePathList"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleGetFiles)

	// get_folder_structure tool
	server.RegisterTool(mcp.Tool{
		Name:        "get_folder_structure",
		Description: "Returns a visual tree representation of the directory structure showing folders and files hierarchically. Use this to understand project layout and organization. For detailed file metadata, use list_context_files instead.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the directory to visualize",
					Examples:    []interface{}{"/home/user/project", "./src", "C:\\Users\\project"},
				},
				"maxDepth": {
					Type:        "integer",
					Description: "Maximum directory depth to traverse. Use 0 for unlimited depth. Lower values improve performance for large directories.",
					Default:     float64(5),
					Minimum:     int64Ptr(0),
					Maximum:     int64Ptr(50),
				},
			},
			Required: []string{"path"},
		},
		Annotations: readOnlyAnnotations(),
	}, handleGetFolderStructure)

	// write_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "write_file",
		Description: "Create a new file or completely overwrite an existing file with new content. Creates parent directories automatically if they do not exist. Warning: this will replace the entire file contents.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to create or overwrite",
					Examples:    []interface{}{"/home/user/project/new-file.txt", "./src/config.json"},
				},
				"content": {
					Type:        "string",
					Description: "The complete content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
		Annotations: writeAnnotations(),
	}, handleWriteFile)

	// create_directory tool
	server.RegisterTool(mcp.Tool{
		Name:        "create_directory",
		Description: "Create a new directory or ensure a directory exists. Creates all necessary parent directories automatically. Safe to call on existing directories.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the directory to create",
					Examples:    []interface{}{"/home/user/project/new-folder", "./src/components/ui"},
				},
			},
			Required: []string{"path"},
		},
		Annotations: writeAnnotations(),
	}, handleCreateDirectory)

	// copy_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "copy_file",
		Description: "Copy a file or directory from source to destination. For directories, performs a recursive copy of all contents. Creates destination parent directories if needed.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"source": {
					Type:        "string",
					Description: "Absolute or relative path to the source file or directory to copy",
					Examples:    []interface{}{"/home/user/project/file.txt", "./src/template"},
				},
				"destination": {
					Type:        "string",
					Description: "Absolute or relative path to the destination location",
					Examples:    []interface{}{"/home/user/project/file-backup.txt", "./src/new-component"},
				},
			},
			Required: []string{"source", "destination"},
		},
		Annotations: writeAnnotations(),
	}, handleCopyFile)

	// move_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "move_file",
		Description: "Move or rename a file or directory from source to destination. The source is removed after successful move. Creates destination parent directories if needed.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"source": {
					Type:        "string",
					Description: "Absolute or relative path to the source file or directory to move",
					Examples:    []interface{}{"/home/user/project/old-name.txt", "./src/legacy"},
				},
				"destination": {
					Type:        "string",
					Description: "Absolute or relative path to the destination location",
					Examples:    []interface{}{"/home/user/project/new-name.txt", "./src/current"},
				},
			},
			Required: []string{"source", "destination"},
		},
		Annotations: writeAnnotations(),
	}, handleMoveFile)

	// delete_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "delete_file",
		Description: "Permanently delete a file or directory from the file system. Warning: this action cannot be undone. For non-empty directories, the recursive flag must be set to true.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file or directory to delete",
					Examples:    []interface{}{"/home/user/project/temp.txt", "./build", "C:\\Users\\project\\cache"},
				},
				"recursive": {
					Type:        "boolean",
					Description: "If true, deletes directories and all their contents recursively. Required for non-empty directories. Use with caution.",
					Default:     false,
				},
			},
			Required: []string{"path"},
		},
		Annotations: destructiveAnnotations(),
	}, handleDeleteFile)

	// modify_file tool
	server.RegisterTool(mcp.Tool{
		Name:        "modify_file",
		Description: "Find and replace text within a file. Supports both literal string matching and regular expressions. Use this for targeted edits rather than rewriting entire files with write_file.",
		InputSchema: mcp.JSONSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"path": {
					Type:        "string",
					Description: "Absolute or relative path to the file to modify",
					Examples:    []interface{}{"/home/user/project/config.json", "./src/index.ts"},
				},
				"find": {
					Type:        "string",
					Description: "Text or regex pattern to search for in the file",
					Examples:    []interface{}{"oldFunctionName", "version: \"1.0.0\"", "import\\s+.*from\\s+'lodash'"},
				},
				"replace": {
					Type:        "string",
					Description: "Text to replace matches with. For regex mode, can include capture group references ($1, $2, etc.)",
					Examples:    []interface{}{"newFunctionName", "version: \"2.0.0\""},
				},
				"all_occurrences": {
					Type:        "boolean",
					Description: "If true, replaces all matching occurrences in the file. If false, replaces only the first match.",
					Default:     true,
				},
				"regex": {
					Type:        "boolean",
					Description: "If true, interprets the 'find' parameter as a regular expression pattern. If false, performs literal string matching.",
					Default:     false,
				},
			},
			Required: []string{"path", "find", "replace"},
		},
		Annotations: writeAnnotations(),
	}, handleModifyFile)
}

func handleListAllowedDirectories(args map[string]interface{}) (*mcp.CallToolResult, error) {
	logger.ToolCall("list_allowed_directories", args)

	result := struct {
		AllowedDirectories []string `json:"allowed_directories"`
		BlockedPatterns    []string `json:"blocked_patterns"`
		Instructions       string   `json:"instructions"`
	}{
		AllowedDirectories: allowedRootDirs,
		BlockedPatterns:    blockedPatterns,
	}

	if len(allowedRootDirs) == 0 {
		result.Instructions = "No directory restrictions. All paths are accessible except those matching blocked patterns."
	} else {
		result.Instructions = "File access is restricted to the listed directories. Paths matching blocked patterns are always denied."
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		logger.Error("list_allowed_directories: failed to marshal result: %v", err)
		return errorResult("Failed to generate result")
	}

	return textResult(string(jsonBytes))
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
	logger.ToolCall("get_files", args)

	filePathList, ok := args["filePathList"].([]interface{})
	if !ok {
		logger.Error("get_files: invalid filePathList parameter")
		return errorResult("Invalid filePathList")
	}

	logger.Debug("get_files: processing %d files", len(filePathList))
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
			logger.Error("get_files: %v", err)
			results[fileName] = map[string]interface{}{
				"error": err.Error(),
			}
			continue
		}

		content, err := files.ReadFile(absPath, DefaultMaxSize)
		if err != nil {
			logger.Error("get_files: failed to read file %q: %v", absPath, err)
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

	logger.Debug("get_files: read %d files, total %d bytes", len(results), totalBytesRead)

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

// parseCommaSeparated splits a comma-separated string into a slice, trimming whitespace
func parseCommaSeparated(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// isBlockedPath checks if the given absolute path matches any blocked pattern
func isBlockedPath(absPath string) bool {
	if len(blockedPatterns) == 0 {
		return false
	}

	// Normalize path separators for matching
	normalizedPath := filepath.ToSlash(absPath)
	baseName := filepath.Base(absPath)

	for _, pattern := range blockedPatterns {
		// Normalize pattern separators
		normalizedPattern := filepath.ToSlash(pattern)

		// Check against the base filename (e.g., ".env" matches "/path/to/.env")
		if matched, _ := doublestar.Match(normalizedPattern, baseName); matched {
			return true
		}

		// Check against path components for directory patterns (e.g., ".aws/*")
		// Walk through path to find matching segments
		parts := strings.Split(normalizedPath, "/")
		for i := range parts {
			// Build subpath from this point
			subpath := strings.Join(parts[i:], "/")
			if matched, _ := doublestar.Match(normalizedPattern, subpath); matched {
				return true
			}
		}
	}
	return false
}

// validatePath checks if the given path is allowed.
// It checks blocked patterns first (deny takes precedence), then root directories.
// Returns the absolute path if valid, or an error if access is denied.
func validatePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check blocked patterns first (deny takes precedence)
	if isBlockedPath(absPath) {
		return "", fmt.Errorf("access denied: path %q matches blocked pattern", path)
	}

	// If no root directory restrictions, allow all paths
	if len(allowedRootDirs) == 0 {
		return absPath, nil
	}

	// Check if path is within ANY allowed root directory
	for _, rootDir := range allowedRootDirs {
		if isSubPath(rootDir, absPath) {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("access denied: path %q is outside allowed directories", path)
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

// Helper functions for tool annotations
func boolPtr(b bool) *bool {
	return &b
}

func int64Ptr(i int64) *int64 {
	return &i
}

func readOnlyAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint: boolPtr(true),
	}
}

func writeAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint: boolPtr(false),
	}
}

func destructiveAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    boolPtr(false),
		DestructiveHint: boolPtr(true),
	}
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
