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

## Installation

### Download Pre-built Binaries

Download the appropriate binary for your platform from the [Releases](https://github.com/JeremyProffitt/go-mcp-file-context-server/releases) page:

| Platform | Architecture | File |
|----------|--------------|------|
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

## Usage with Claude Desktop

Add to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server"
    }
  }
}
```

On Windows:
```json
{
  "mcpServers": {
    "file-context": {
      "command": "C:\\path\\to\\go-mcp-file-context-server-windows-amd64.exe"
    }
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

## Development

```bash
# Get dependencies
go mod download

# Run tests
go test -v ./...

# Build for current platform
go build -v .

# Build for all platforms
GOOS=darwin GOARCH=arm64 go build -o bin/darwin-arm64/go-mcp-file-context-server .
GOOS=darwin GOARCH=amd64 go build -o bin/darwin-amd64/go-mcp-file-context-server .
GOOS=linux GOARCH=amd64 go build -o bin/linux-amd64/go-mcp-file-context-server .
GOOS=linux GOARCH=arm64 go build -o bin/linux-arm64/go-mcp-file-context-server .
GOOS=windows GOARCH=amd64 go build -o bin/windows-amd64/go-mcp-file-context-server.exe .
```

## License

MIT

## Credits

This is a Go port of the original [mcp-file-context-server](https://github.com/bsmi021/mcp-file-context-server) by [@bsmi021](https://github.com/bsmi021).
