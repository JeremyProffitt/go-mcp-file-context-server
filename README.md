# go-mcp-file-context-server

A Go port of the [MCP File Context Server](https://github.com/bsmi021/mcp-file-context-server) - a Model Context Protocol (MCP) server that provides file system context to Large Language Models (LLMs).

## Features

- **File Read Operations**
  - Read file and directory contents with metadata
  - List files with detailed metadata
  - Recursive directory traversal
  - File type filtering
  - Batch file retrieval
  - Folder structure tree generation

- **File Write Operations**
  - Create new files or overwrite existing files
  - Create directories (including nested paths)
  - Copy files and directories
  - Move/rename files and directories
  - Delete files and directories
  - Find and replace text (with regex support)

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
  -root-dir <paths>   Root directories to restrict file access (comma-separated)
                      Default: no restriction (full filesystem access)

  -blocked-patterns <patterns>
                      Patterns to block access to (comma-separated globs)
                      Default: .aws/*,.env,.mcp_env

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
| `MCP_ROOT_DIR` | Restrict file access to these directories (comma-separated) | No restriction |
| `MCP_BLOCKED_PATTERNS` | Block access to files matching these patterns (comma-separated globs) | `.aws/*,.env,.mcp_env` |
| `MCP_LOG_DIR` | Directory for log files | `~/go-mcp-file-context-server/logs` |
| `MCP_LOG_LEVEL` | Log level (off, error, warn, info, access, debug) | `info` |

**Note:** Setting `MCP_BLOCKED_PATTERNS` to an empty string disables all file blocking.

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

**Basic setup (restrict to project directory):**

```json
{
  "mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "args": [
        "-root-dir", "/Users/username/projects/myproject",
        "-log-level", "access"
      ]
    }
  }
}
```

**With environment variables:**

```json
{
  "mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "env": {
        "MCP_ROOT_DIR": "/Users/username/projects,/Users/username/work",
        "MCP_BLOCKED_PATTERNS": ".aws/*,.env,.mcp_env,secrets/*",
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
      "args": [
        "-root-dir", "C:\\Users\\username\\projects\\myproject",
        "-log-level", "info"
      ]
    }
  }
}
```

**Full filesystem access (no restriction):**

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

### VS Code with Claude Extension

Add to your VS Code settings (`settings.json`):

```json
{
  "claude.mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "args": [
        "-root-dir", "/path/to/project",
        "-log-level", "access"
      ]
    }
  }
}
```

Or in VS Code workspace settings (`.vscode/settings.json`) with workspace-relative paths:

```json
{
  "claude.mcpServers": {
    "file-context": {
      "command": "${workspaceFolder}/bin/go-mcp-file-context-server",
      "args": ["-log-level", "debug"],
      "env": {
        "MCP_ROOT_DIR": "${workspaceFolder}",
        "MCP_LOG_DIR": "${workspaceFolder}/logs"
      }
    }
  }
}
```

### Claude Code CLI

For local workspace configuration with Claude Code CLI, add a `.claude/mcp.json` file to your project root:

**Windows example:**

```json
{
  "mcpServers": {
    "file-context": {
      "command": "${workspaceFolder}/go-mcp-file-context-server.exe",
      "args": [
        "-root-dir", "${workspaceFolder}",
        "-log-dir", "${workspaceFolder}\\logs",
        "-log-level", "debug"
      ]
    }
  }
}
```

**macOS/Linux example:**

```json
{
  "mcpServers": {
    "file-context": {
      "command": "${workspaceFolder}/go-mcp-file-context-server",
      "args": [
        "-root-dir", "${workspaceFolder}",
        "-log-dir", "${workspaceFolder}/logs",
        "-log-level", "debug"
      ]
    }
  }
}
```

### Continue.dev

> **Note:** MCP tools can only be used in **agent mode** in Continue.dev. See [CONTINUE.md](CONTINUE.md) for detailed integration guide.

Add a YAML file in `.continue/mcpServers/` at your workspace root:

**Example: `.continue/mcpServers/file-context.yaml`**

```yaml
name: File Context Server
version: 0.0.1
schema: v1
mcpServers:
  - name: file-context
    command: /path/to/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/username/projects/myproject"
      - "-log-level"
      - "access"
```

**Windows example:**

```yaml
name: File Context Server
version: 0.0.1
schema: v1
mcpServers:
  - name: file-context
    command: C:/path/to/go-mcp-file-context-server.exe
    args:
      - "-root-dir"
      - "C:/Users/username/projects/myproject"
      - "-log-level"
      - "info"
```

### Global Configuration

**VS Code Extension:** `~/.continue/config.yaml`

```yaml
name: Local Config
version: 1.0.0
schema: v1

models: []

mcpServers:
  - name: file-context
    command: /path/to/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/username/projects"
      - "-log-level"
      - "debug"
```

**CLI:** `~/.continue/continue.yaml`

```yaml
mcpServers:
  - name: file-context
    command: /path/to/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/username/projects"
      - "-log-level"
      - "debug"
```

### Project-Local Configuration

Add a `.continuerc.yaml` file to your project root:

```yaml
mcpServers:
  - name: file-context
    command: ./go-mcp-file-context-server
    args:
      - "-root-dir"
      - "."
      - "-log-dir"
      - "./logs"
      - "-log-level"
      - "debug"
```

**Note:** Using `-root-dir "."` restricts access to the current working directory.

## Tool Reference

This section provides a categorized overview of all available tools to help LLMs quickly identify the right tool for each task.

### Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| **Discovery** | `list_context_files`, `get_folder_structure` | Understand project structure before reading files |
| **Reading** | `read_context`, `getFiles` | Retrieve file contents |
| **Search** | `search_context` | Find patterns across files |
| **Analysis** | `analyze_code`, `generate_outline` | Understand code quality and structure |
| **Writing** | `write_file`, `create_directory`, `copy_file`, `move_file`, `delete_file`, `modify_file` | Modify filesystem |
| **Utility** | `cache_stats`, `get_chunk_count` | Performance and chunking info |

### Tool Selection Guide

**When to use `list_context_files` vs `read_context` vs `get_folder_structure`:**

| Scenario | Best Tool | Why |
|----------|-----------|-----|
| "What files exist in this project?" | `get_folder_structure` | Returns tree view of entire directory structure efficiently |
| "Show me all .go files in src/" | `list_context_files` | Returns filtered file list with metadata (size, modified date) |
| "Read the contents of main.go" | `read_context` | Returns actual file content |
| "What's in the config directory?" | `list_context_files` | Shows directory contents with details |
| "Give me an overview of the codebase" | `get_folder_structure` + `list_context_files` | Structure first, then targeted file lists |

**Decision flowchart for file operations:**

1. **Need to understand project layout?** -> `get_folder_structure` (maxDepth: 3-5)
2. **Need to find specific file types?** -> `list_context_files` with `fileTypes` filter
3. **Need to read file contents?** -> `read_context` for single file, `getFiles` for multiple
4. **Need to find code patterns?** -> `search_context` with regex pattern
5. **Need to understand code structure?** -> `generate_outline` for classes/functions/imports
6. **Need code quality metrics?** -> `analyze_code` for complexity and issues

### Common Workflows

**Workflow 1: Exploring a new codebase**
```
1. get_folder_structure(path: ".", maxDepth: 3)     # Understand layout
2. list_context_files(path: ".", recursive: true)   # See all files with sizes
3. read_context(path: "README.md")                  # Read documentation
4. generate_outline(path: "src/main.go")            # Understand entry point
```

**Workflow 2: Finding and fixing a bug**
```
1. search_context(pattern: "error|bug|TODO", path: "src/", contextLines: 3)
2. read_context(path: "src/problematic_file.go")
3. modify_file(path: "src/problematic_file.go", find: "buggy_code", replace: "fixed_code")
```

**Workflow 3: Code review and analysis**
```
1. analyze_code(path: "src/", recursive: true)      # Get complexity metrics
2. generate_outline(path: "src/main.go")            # See code structure
3. search_context(pattern: "TODO|FIXME|HACK")       # Find tech debt
```

**Workflow 4: Batch file reading**
```
1. list_context_files(path: "src/", fileTypes: ["go"])  # Get file list
2. getFiles(filePathList: [{"fileName": "src/a.go"}, {"fileName": "src/b.go"}])
```

**Workflow 5: Large file handling**
```
1. get_chunk_count(path: "large_file.log")          # Check chunk count
2. read_context(path: "large_file.log", chunkNumber: 0)  # Read first chunk
3. read_context(path: "large_file.log", chunkNumber: 1)  # Read next chunk
```

---

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

### write_file
Create a new file or overwrite an existing file with content.

```json
{
  "path": "./src/newfile.go",
  "content": "package main\n\nfunc main() {}\n"
}
```

### create_directory
Create a new directory (including parent directories if needed).

```json
{
  "path": "./src/components/new-feature"
}
```

### copy_file
Copy a file or directory from source to destination.

```json
{
  "source": "./src/template.go",
  "destination": "./src/newfile.go"
}
```

### move_file
Move or rename a file or directory.

```json
{
  "source": "./src/oldname.go",
  "destination": "./src/newname.go"
}
```

### delete_file
Delete a file or directory from the file system.

```json
{
  "path": "./src/deprecated.go",
  "recursive": false
}
```

For directories with contents, set `recursive: true`.

### modify_file
Find and replace text in a file. Supports regex patterns.

```json
{
  "path": "./src/main.go",
  "find": "oldFunction",
  "replace": "newFunction",
  "all_occurrences": true,
  "regex": false
}
```

With regex:

```json
{
  "path": "./src/main.go",
  "find": "func (\\w+)\\(\\)",
  "replace": "function $1()",
  "all_occurrences": true,
  "regex": true
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

## Global Environment File

All go-mcp servers support loading environment variables from `~/.mcp_env`. This provides a central location to configure credentials and settings, especially useful on macOS where GUI applications don't inherit shell environment variables from `.zshrc` or `.bashrc`.

### File Format

Create `~/.mcp_env` with KEY=VALUE pairs:

```bash
# ~/.mcp_env - MCP Server Environment Variables

# File Context Server Configuration
MCP_ROOT_DIR=~/projects,~/work
MCP_BLOCKED_PATTERNS=.aws/*,.env,.mcp_env
MCP_LOG_DIR=~/mcp-logs
MCP_LOG_LEVEL=info
```

### Features

- Lines starting with `#` are treated as comments
- Empty lines are ignored
- Values can be quoted with single or double quotes
- **Existing environment variables are NOT overwritten** (env vars take precedence)
- Paths with `~` are automatically expanded to your home directory
- Multiple root directories can be specified with comma separation
- Blocked patterns support glob syntax (e.g., `.aws/*`, `secrets/**`)

### Path Expansion

All path-related settings support `~` expansion:

```bash
MCP_ROOT_DIR=~/projects/my-app,~/work/other-project
MCP_LOG_DIR=~/logs/file-context
```

This works in the `~/.mcp_env` file, environment variables, and command-line flags.

## Log File Location

By default, log files are stored in:

| Platform | Default Location |
|----------|------------------|
| macOS/Linux | `~/go-mcp-file-context-server/logs/` |
| Windows | `%USERPROFILE%\go-mcp-file-context-server\logs\` |

Log files are named with the format: `go-mcp-file-context-server-YYYY-MM-DD.log`

When `MCP_LOG_DIR` is set or `-log-dir` flag is used, logs are automatically placed in a subfolder named after the binary. This allows multiple MCP servers to share the same log directory:

```
MCP_LOG_DIR=/var/log/mcp
  └── go-mcp-file-context-server/
      └── go-mcp-file-context-server-2025-01-15.log
```

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
