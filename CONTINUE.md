# Continue.dev Integration Guide

This guide covers integrating the go-mcp-file-context-server with Continue.dev for both the VS Code extension and the `cn` CLI.

> **Important:** MCP tools can only be used in **agent mode** in Continue.dev.

## Configuration Files Overview

Continue.dev uses **different configuration files** depending on the client:

| File | Location | Used By |
|------|----------|---------|
| `config.yaml` | `~/.continue/config.yaml` | **VS Code Extension** |
| `continue.yaml` | `~/.continue/continue.yaml` | **`cn` CLI** |
| `.continuerc.yaml` | Project root | Project-local overrides (both) |

> **Important:** On Windows, `~` refers to `%USERPROFILE%` (e.g., `C:\Users\YourName`)

---

## VS Code Extension Setup

### Step 1: Locate the Config File

**Windows:**
```
C:\Users\YourName\.continue\config.yaml
```

**macOS/Linux:**
```
~/.continue/config.yaml
```

### Step 2: Add MCP Server Configuration

Edit `config.yaml` and add the `mcpServers` section:

**Windows Example:**
```yaml
name: Local Config
version: 1.0.0
schema: v1

models: []

mcpServers:
  - name: file-context
    command: "C:/dev/go-mcp-file-context-server/go-mcp-file-context-server.exe"
    args:
      - "-root-dir"
      - "C:/dev"
      - "-log-level"
      - "debug"
```

**macOS/Linux Example:**
```yaml
name: Local Config
version: 1.0.0
schema: v1

models: []

mcpServers:
  - name: file-context
    command: /usr/local/bin/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/username/projects"
      - "-log-level"
      - "debug"
```

> **Note:** On Windows, use forward slashes (`/`) in paths for YAML compatibility.

### Step 3: Reload VS Code

After editing the config, reload VS Code:

1. Press `Ctrl+Shift+P` (Windows/Linux) or `Cmd+Shift+P` (macOS)
2. Type "Developer: Reload Window" and press Enter

### Troubleshooting VS Code Extension

**Check Continue Output Panel:**
1. Press `Ctrl+Shift+P` -> "Continue: Focus on Continue Output"
2. Look for MCP-related errors or connection issues

**Common Issues:**

| Issue | Solution |
|-------|----------|
| MCP server not registering | Ensure you edited `config.yaml`, not `continue.yaml` |
| MCP tools not available | MCP only works in **agent mode** |
| Binary not found | Use absolute path to the executable |
| Permission denied | Ensure binary is executable (`chmod +x` on macOS/Linux) |
| Config syntax error | Validate YAML syntax (indentation matters) |
| Path issues on Windows | Use forward slashes (`/`) instead of backslashes |

---

## CLI (`cn`) Setup

### Step 1: Locate the Config File

**Windows:**
```
C:\Users\YourName\.continue\continue.yaml
```

**macOS/Linux:**
```
~/.continue/continue.yaml
```

### Step 2: Add MCP Server Configuration

Edit `continue.yaml` and add the `mcpServers` section:

**Windows Example:**
```yaml
mcpServers:
  - name: file-context
    command: "C:/dev/go-mcp-file-context-server/go-mcp-file-context-server.exe"
    args:
      - "-root-dir"
      - "C:/dev"
      - "-log-level"
      - "debug"
```

**macOS/Linux Example:**
```yaml
mcpServers:
  - name: file-context
    command: /usr/local/bin/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/username/projects"
      - "-log-level"
      - "debug"
```

---

## Project-Local Configuration

For project-specific settings, create a `.continuerc.yaml` file in your project root. This works with both VS Code and `cn` CLI.

**Example `.continuerc.yaml`:**
```yaml
mcpServers:
  - name: file-context
    command: ./go-mcp-file-context-server.exe  # Relative to project root
    args:
      - "-root-dir"
      - "."
      - "-log-dir"
      - "./logs"
      - "-log-level"
      - "debug"
```

---

## Configuration Options

### Available Command-Line Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `-root-dir <path>` | Restrict file access to this directory | No restriction |
| `-log-dir <path>` | Directory for log files | `~/go-mcp-file-context-server/logs` |
| `-log-level <level>` | Logging verbosity | `info` |

### Log Levels

| Level | Description |
|-------|-------------|
| `off` | Disable all logging |
| `error` | Errors only |
| `warn` | Warnings and errors |
| `info` | General information (default) |
| `access` | File access operations (includes bytes read/written) |
| `debug` | Detailed debugging information |

---

## Log File Location

If `-log-dir` is not specified, logs are written to:

**Windows:**
```
C:\Users\YourName\go-mcp-file-context-server\logs\go-mcp-file-context-server-YYYY-MM-DD.log
```

**macOS/Linux:**
```
~/go-mcp-file-context-server/logs/go-mcp-file-context-server-YYYY-MM-DD.log
```

---

## Full Configuration Examples

### VS Code Extension with All Options (`config.yaml`)

```yaml
name: Local Config
version: 1.0.0
schema: v1

models:
  - id: claude-3.5-sonnet
    provider: anthropic
    model: claude-3.5-sonnet
    capabilities:
      - chat
      - edit

mcpServers:
  - name: file-context
    command: "C:/dev/go-mcp-file-context-server/go-mcp-file-context-server.exe"
    args:
      - "-root-dir"
      - "C:/dev/myproject"
      - "-log-dir"
      - "C:/logs/mcp"
      - "-log-level"
      - "access"
```

### CLI with All Options (`continue.yaml`)

```yaml
version: 2

models:
  - id: gpt-4
    provider: openai
    model: gpt-4
    default: true
    capabilities:
      - chat
      - edit
      - autocomplete

mcpServers:
  - name: file-context
    command: /usr/local/bin/go-mcp-file-context-server
    args:
      - "-root-dir"
      - "/home/user/projects/myapp"
      - "-log-dir"
      - "/var/log/mcp"
      - "-log-level"
      - "access"
```

---

## Verifying the Integration

### Check if MCP Server is Running

1. Look for log files in the log directory
2. The startup log should show:
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

### Test MCP Tools

Once connected, the following tools should be available (in agent mode):
- `list_context_files` - List files in a directory
- `read_context` - Read file contents
- `search_context` - Search for patterns in files
- `analyze_code` - Analyze code complexity and quality
- `generate_outline` - Generate code outline
- `cache_stats` - View cache statistics
- `get_chunk_count` - Get chunk count for large files
- `getFiles` - Batch retrieve multiple files
- `get_folder_structure` - Get directory tree structure

---

## Migration from Legacy Format

If you have an older configuration using `experimental.modelContextProtocolServers`, migrate to the new `mcpServers` format:

**Old format (deprecated):**
```yaml
experimental:
  modelContextProtocolServers:
    - name: file-context
      transport:
        type: stdio
        command: /path/to/binary
        args:
          - "-arg1"
          - "value1"
```

**New format:**
```yaml
mcpServers:
  - name: file-context
    command: /path/to/binary
    args:
      - "-arg1"
      - "value1"
```
