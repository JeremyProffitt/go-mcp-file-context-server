# go-mcp-file-context-server

A Go port of the [MCP File Context Server](https://github.com/bsmi021/mcp-file-context-server) - a Model Context Protocol (MCP) server that provides file system context to Large Language Models (LLMs).

## Features

- **File Operations**
  - Read file and directory contents with metadata
  - List files with detailed metadata
  - Real-time file watching and cache invalidation
  - Recursive directory traversal
  - File type filtering
  - Batch file retrieval

- **Code Analysis**
  - Cyclomatic complexity calculation
  - Dependency/import extraction
  - Code outline generation (classes, functions, imports)
  - Quality metrics:
    - Duplicate lines detection
    - Long lines detection (>100 characters)
    - Complex function identification
    - Line counts (total, non-empty, comments)

- **Smart Caching**
  - LRU (Least Recently Used) caching strategy
  - Configurable TTL (Time-To-Live)
  - Cache statistics and performance metrics

- **Advanced Search**
  - Regex pattern matching
  - Context-aware results with configurable surrounding lines
  - File type filtering
  - Multi-pattern search support

- **Comprehensive Logging**
  - File access logging (file names and bytes, never content)
  - Detailed startup information
  - Configurable log levels
  - Automatic log directory creation

## Installation

### Download Pre-built Binaries

Download the appropriate binary for your platform from the [Releases](https://github.com/JeremyProffitt/go-mcp-file-context-server/releases) page:

| Platform | Architecture | File |
|----------|--------------|------|
| macOS | Universal (Intel + Apple Silicon) | `go-mcp-file-context-server-darwin-universal` |
| macOS | Apple Silicon (M1/M2/M3) | `go-mcp-file-context-server-darwin-arm64` |
| macOS | Intel | `go-mcp-file-context-server-darwin-amd64` |
| Linux | x86_64 | `go-mcp-file-context-server-linux-amd64` |
| Linux | ARM64 | `go-mcp-file-context-server-linux-arm64` |
| Windows | x86_64 | `go-mcp-file-context-server-windows-amd64.exe` |

### Build from Source

```bash
# Clone the repository
git clone https://github.com/JeremyProffitt/go-mcp-file-context-server.git
cd go-mcp-file-context-server

# Build
go build -o go-mcp-file-context-server .

# Run
./go-mcp-file-context-server
```

## Configuration

### Command Line Options

```
Usage: go-mcp-file-context-server [OPTIONS]

Options:
  -log-dir <path>     Directory for log files
                      Default: ~/go-mcp-file-context-server/logs

  -log-level <level>  Log level: off, error, warn, info, access, debug
                      Default: info

  -version            Show version information

  -help               Show help message
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_LOG_DIR` | Directory for log files | `~/go-mcp-file-context-server/logs` |
| `MCP_LOG_LEVEL` | Log level (off, error, warn, info, access, debug) | `info` |

### Configuration Priority

Configuration values are resolved in the following order (first match wins):
1. Command line flags
2. Environment variables
3. Default values

### Log Levels

| Level | Description |
|-------|-------------|
| `off` | Disable all logging |
| `error` | Log errors only |
| `warn` | Log warnings and errors |
| `info` | Log general information, warnings, and errors (default) |
| `access` | Log file access operations (includes file names and bytes read/written) |
| `debug` | Log detailed debugging information |

**Security Note:** Logging NEVER captures actual file content. Only file names, paths, and byte counts are logged.

### Default Settings

| Setting | Default Value | Description |
|---------|---------------|-------------|
| Max File Size | 10 MB | Maximum file size for single read operations |
| Cache Size | 500 entries | LRU cache capacity |
| Cache TTL | 5 minutes | Time before cached entries expire |
| Chunk Size | 64 KB | Size of each chunk for large files |

### Default Ignore Patterns

The following patterns are automatically ignored during file operations:

```
.git, node_modules, .vscode, .idea, __pycache__, .DS_Store,
*.pyc, .env, dist, build, coverage, .next, .nuxt, vendor, .cache
```

## MCP Server Setup

### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "args": ["-log-level", "access"]
    }
  }
}
```

With environment variables:

```json
{
  "mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "env": {
        "MCP_LOG_DIR": "/var/log/mcp-file-context",
        "MCP_LOG_LEVEL": "debug"
      }
    }
  }
}
```

**Windows example:**

```json
{
  "mcpServers": {
    "file-context": {
      "command": "C:\\path\\to\\go-mcp-file-context-server-windows-amd64.exe",
      "args": ["-log-level", "info"]
    }
  }
}
```

### VS Code with Claude Extension

Add to your VS Code settings (`settings.json`):

```json
{
  "claude.mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "args": ["-log-level", "access"]
    }
  }
}
```

Or in VS Code workspace settings (`.vscode/settings.json`):

```json
{
  "claude.mcpServers": {
    "file-context": {
      "command": "${workspaceFolder}/bin/go-mcp-file-context-server",
      "args": ["-log-level", "debug"],
      "env": {
        "MCP_LOG_DIR": "${workspaceFolder}/logs"
      }
    }
  }
}
```

### Continue.dev

Add to your Continue configuration file:

**Location:** `~/.continue/config.json` (macOS/Linux) or `%USERPROFILE%\.continue\config.json` (Windows)

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-file-context-server",
          "args": ["-log-level", "access"]
        }
      }
    ]
  }
}
```

With environment variables:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-file-context-server",
          "args": ["-log-dir", "/custom/log/path", "-log-level", "debug"]
        }
      }
    ]
  }
}
```

**Windows example:**

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "C:\\path\\to\\go-mcp-file-context-server-windows-amd64.exe",
          "args": ["-log-level", "info"]
        }
      }
    ]
  }
}
```

### Multiple Projects Configuration

For Continue.dev with project-specific settings, you can use workspace-relative paths:

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "transport": {
          "type": "stdio",
          "command": "go-mcp-file-context-server",
          "args": [
            "-log-dir", "./logs/mcp",
            "-log-level", "access"
          ]
        }
      }
    ]
  }
}
```

## Available Tools

### list_context_files
Lists files in a directory with detailed metadata.

```json
{
  "path": "./src",
  "recursive": true,
  "includeHidden": false,
  "fileTypes": ["go", "ts", "py"]
}
```

### read_context
Reads file or directory contents with metadata and caching.

```json
{
  "path": "./src/main.go",
  "maxSize": 10485760,
  "encoding": "utf8",
  "recursive": true,
  "fileTypes": ["go"],
  "chunkNumber": 0
}
```

### search_context
Searches for patterns in files with context lines.

```json
{
  "pattern": "func.*Handler",
  "path": "./src",
  "recursive": true,
  "fileTypes": ["go"],
  "contextLines": 3,
  "maxResults": 100
}
```

### analyze_code
Analyzes code files for complexity, dependencies, and quality metrics.

```json
{
  "path": "./src",
  "recursive": true,
  "fileTypes": ["go", "ts"]
}
```

### generate_outline
Generates a code outline showing classes, functions, and imports.

```json
{
  "path": "./src/main.go"
}
```

### cache_stats
Returns cache statistics and performance metrics.

```json
{
  "detailed": true
}
```

### get_chunk_count
Gets the total number of chunks for a file or directory.

```json
{
  "path": "./large_file.txt",
  "chunkSize": 65536
}
```

### getFiles
Batch retrieve multiple files at once.

```json
{
  "filePathList": [
    {"fileName": "./src/main.go"},
    {"fileName": "./pkg/utils.go"}
  ]
}
```

### get_folder_structure
Returns a tree representation of the folder structure.

```json
{
  "path": "./src",
  "maxDepth": 5
}
```

## Supported Languages for Code Analysis

- Go
- TypeScript/JavaScript
- Python
- Java
- Rust
- C/C++
- Ruby
- PHP

## Log File Location

By default, log files are stored in:

| Platform | Default Location |
|----------|------------------|
| macOS/Linux | `~/go-mcp-file-context-server/logs/` |
| Windows | `%USERPROFILE%\go-mcp-file-context-server\logs\` |

Log files are named with the format: `go-mcp-file-context-server-YYYY-MM-DD.log`

### Sample Log Output

```
[2025-01-15T10:30:45.123Z] [INFO] ========================================
[2025-01-15T10:30:45.123Z] [INFO] SERVER STARTUP
[2025-01-15T10:30:45.123Z] [INFO] ========================================
[2025-01-15T10:30:45.123Z] [INFO] Application: go-mcp-file-context-server
[2025-01-15T10:30:45.123Z] [INFO] Version: 1.0.0
[2025-01-15T10:30:45.123Z] [INFO] Go Version: go1.21.0
[2025-01-15T10:30:45.123Z] [INFO] OS: darwin
[2025-01-15T10:30:45.124Z] [INFO] Architecture: arm64
[2025-01-15T10:30:45.124Z] [INFO] Log Level: ACCESS
[2025-01-15T10:30:45.200Z] [INFO] TOOL_CALL tool="read_context" args=[path]
[2025-01-15T10:30:45.205Z] [ACCESS] FILE_READ path="/Users/dev/project/main.go" bytes=4521
```

## Development

### Build for Current Platform

```bash
go build -v .
```

### Build for All Platforms

```bash
# macOS Universal Binary (Intel + Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o bin/go-mcp-file-context-server-darwin-arm64 .
GOOS=darwin GOARCH=amd64 go build -o bin/go-mcp-file-context-server-darwin-amd64 .
lipo -create -output bin/go-mcp-file-context-server-darwin-universal \
  bin/go-mcp-file-context-server-darwin-arm64 \
  bin/go-mcp-file-context-server-darwin-amd64

# Linux
GOOS=linux GOARCH=amd64 go build -o bin/go-mcp-file-context-server-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o bin/go-mcp-file-context-server-linux-arm64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/go-mcp-file-context-server-windows-amd64.exe .
```

### Run Tests

```bash
go test -v ./...
```

### Build Script

A build script is provided for convenience:

```bash
./build.sh
```

This will create binaries for all platforms in the `bin/` directory.

## Troubleshooting

### Logs Not Being Created

1. Ensure the log directory exists and is writable
2. Check that the `-log-level` is not set to `off`
3. Verify environment variables are being passed correctly

### Server Not Starting

1. Check for port conflicts or permission issues
2. Verify the binary has execute permissions: `chmod +x go-mcp-file-context-server`
3. Run with `-log-level debug` for detailed startup information

### Files Not Being Read

1. Check file permissions
2. Verify the path is correct (use absolute paths when possible)
3. Check if the file is in an ignored pattern (`.git`, `node_modules`, etc.)
4. Verify file size doesn't exceed `maxSize` parameter

## License

MIT

## Credits

This is a Go port of the original [mcp-file-context-server](https://github.com/bsmi021/mcp-file-context-server) by [@bsmi021](https://github.com/bsmi021).
