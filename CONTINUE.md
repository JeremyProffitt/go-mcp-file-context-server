# Continue.dev Integration Guide

This guide covers integrating the go-mcp-file-context-server with Continue.dev for both the VS Code extension and the `cn` CLI.

> **Important:** MCP tools can only be used in **agent mode** in Continue.dev.

## LLM Quick Reference

When using this MCP server in Continue.dev agent mode, remember:

| Goal | Tool to Use | Example |
|------|-------------|---------|
| Explore project | `get_folder_structure` | `get_folder_structure(path: ".", maxDepth: 3)` |
| Find specific files | `list_context_files` | `list_context_files(path: "src", fileTypes: ["ts", "tsx"])` |
| Read a file | `read_context` | `read_context(path: "src/main.ts")` |
| Search code | `search_context` | `search_context(pattern: "useState", path: "src")` |
| Edit a file | `modify_file` | `modify_file(path: "file.ts", find: "old", replace: "new")` |

---

## YAML Configuration

Create a YAML file in `.continue/mcpServers/` at your workspace root.

### Example: `.continue/mcpServers/file-context.yaml`

**Windows:**
```yaml
name: File Context Server
version: 0.0.1
schema: v1
mcpServers:
  - name: file-context
    type: stdio
    command: C:/dev/go-mcp-file-context-server/go-mcp-file-context-server.exe
    args:
      - "-root-dir"
      - "C:/dev/myproject"
      - "-log-level"
      - "debug"
```

**macOS/Linux:**
```yaml
name: File Context Server
version: 0.0.1
schema: v1
mcpServers:
  - name: file-context
    type: stdio
    command: /usr/local/bin/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/user/projects"
      - "-log-level"
      - "debug"
```

---

## JSON Configuration

Continue.dev automatically recognizes JSON MCP configs from other tools.

### Example: `.continue/mcpServers/mcp.json`

**Windows:**
```json
{
  "mcpServers": {
    "file-context": {
      "command": "C:/dev/go-mcp-file-context-server/go-mcp-file-context-server.exe",
      "args": ["-root-dir", "C:/dev", "-log-level", "debug"]
    }
  }
}
```

**macOS/Linux:**
```json
{
  "mcpServers": {
    "file-context": {
      "command": "/usr/local/bin/go-mcp-file-context-server",
      "args": ["-root-dir", "/home/user/projects", "-log-level", "debug"]
    }
  }
}
```

---

## Available Tools

Once connected, the following tools are available in agent mode:

### Read Operations
| Tool | Description |
|------|-------------|
| `list_context_files` | List files in a directory with metadata |
| `read_context` | Read file or directory contents with caching |
| `search_context` | Search for patterns in files with context |
| `analyze_code` | Analyze code complexity and quality metrics |
| `generate_outline` | Generate code outline (classes, functions, imports) |
| `cache_stats` | View cache statistics and performance |
| `get_chunk_count` | Get chunk count for large files |
| `getFiles` | Batch retrieve multiple files |
| `get_folder_structure` | Get directory tree structure |

### Write Operations
| Tool | Description |
|------|-------------|
| `write_file` | Create or overwrite a file with new content |
| `create_directory` | Create a new directory (including parents) |
| `copy_file` | Copy a file or directory |
| `move_file` | Move or rename a file or directory |
| `delete_file` | Delete a file or directory |
| `modify_file` | Find and replace text (supports regex) |

---

## Command-Line Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `-root-dir <path>` | Restrict file access to this directory | No restriction |
| `-log-dir <path>` | Directory for log files | See Log File Location |
| `-log-level <level>` | Logging verbosity | `info` |

### Log Levels

| Level | Description |
|-------|-------------|
| `off` | Disable all logging |
| `error` | Errors only |
| `warn` | Warnings and errors |
| `info` | General information (default) |
| `access` | File access operations |
| `debug` | Detailed debugging information |

---

## Log File Location

**Windows:**
```
C:\Users\YourName\go-mcp-file-context-server\logs\go-mcp-file-context-server-YYYY-MM-DD.log
```

**macOS/Linux:**
```
/home/username/go-mcp-file-context-server/logs/go-mcp-file-context-server-YYYY-MM-DD.log
```

---

## Verifying the Integration

### Check Server Logs

The startup log should show:
```
[INFO] ========================================
[INFO] SERVER STARTUP
[INFO] ========================================
[INFO] Application: go-mcp-file-context-server
[INFO] Version: 1.0.0
...
[INFO] CONFIGURATION (value [source])
[INFO] ----------------------------------------
[INFO] Log Directory: /path/to/logs [default]
[INFO] Log Level: debug [flag]
[INFO] Root Directory: /path/to/root [flag]
```

### Reload VS Code

After editing configuration:
1. Press `Ctrl+Shift+P` (Windows/Linux) or `Cmd+Shift+P` (macOS)
2. Type "Developer: Reload Window" and press Enter

### Check Continue Output

1. Press `Ctrl+Shift+P` → "Continue: Focus on Continue Output"
2. Look for MCP-related connection messages

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| MCP server not registering | Ensure correct config file for your client (VS Code vs CLI) |
| MCP tools not available | MCP only works in **agent mode** |
| Binary not found | Use absolute path to the executable |
| Permission denied | Ensure binary is executable (`chmod +x` on macOS/Linux) |
| Config syntax error | Validate YAML syntax (indentation matters) |
| Path issues on Windows | Use forward slashes (`/`) in YAML paths |
