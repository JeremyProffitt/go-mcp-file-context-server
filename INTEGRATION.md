# MCP Client Integration Guide

This guide explains how to configure MCP clients (Claude Code and Continue.dev) to connect to the go-mcp-file-context-server running in HTTP mode, including authentication configuration.

## LLM Tool Selection Guide

When connected to this server, use these tools based on your task:

### Discovery Phase (Always Start Here)
| Tool | When to Use | Key Parameters |
|------|-------------|----------------|
| `get_folder_structure` | First look at any project | `maxDepth: 3-5` for large projects |
| `list_context_files` | See file details, filter by type | `fileTypes: ["go", "py"]` |
| `list_allowed_directories` | Check which paths are accessible | None required |

### Reading Files
| Tool | When to Use | Key Parameters |
|------|-------------|----------------|
| `read_context` | Read single file | `chunkNumber` for large files |
| `getFiles` | Read multiple files at once | `filePathList` array |
| `get_chunk_count` | Check if file needs chunking | `path` to file |

### Searching and Analysis
| Tool | When to Use | Key Parameters |
|------|-------------|----------------|
| `search_context` | Find patterns in code | `pattern` (regex), `contextLines` |
| `analyze_code` | Get code quality metrics | `recursive: true` for directories |
| `generate_outline` | See classes/functions/imports | Single file `path` |

### Writing and Modifying
| Tool | When to Use | Key Parameters |
|------|-------------|----------------|
| `write_file` | Create or overwrite file | `path`, `content` |
| `modify_file` | Find and replace text | `find`, `replace`, `regex: true` for patterns |
| `create_directory` | Create new folder | `path` (creates parents too) |
| `copy_file` / `move_file` | Copy or rename files | `source`, `destination` |
| `delete_file` | Remove file or directory | `recursive: true` for directories |

---

## Authentication Overview

When running in HTTP mode with authentication enabled (via `MCP_AUTH_TOKEN` environment variable), all requests must include the `X-MCP-Auth-Token` header with the configured token value.

## Claude Code Integration

### Configuration Location

Claude Code configuration is stored in:
- **macOS/Linux**: `~/.claude/claude_code_config.json`
- **Windows**: `%USERPROFILE%\.claude\claude_code_config.json`

### HTTP Mode Configuration

```json
{
  "mcpServers": {
    "file-context": {
      "type": "http",
      "url": "http://your-alb-url:3000",
      "headers": {
        "X-MCP-Auth-Token": "your-secure-auth-token"
      }
    }
  }
}
```

### Configuration with Environment Variable

```json
{
  "mcpServers": {
    "file-context": {
      "type": "http",
      "url": "http://your-alb-url:3000",
      "headers": {
        "X-MCP-Auth-Token": "${MCP_FILE_CONTEXT_TOKEN}"
      }
    }
  }
}
```

### Local Development (stdio mode)

```json
{
  "mcpServers": {
    "file-context": {
      "command": "/path/to/go-mcp-file-context-server",
      "args": ["--root-dir", "/path/to/allowed/directory"],
      "env": {}
    }
  }
}
```

## Continue.dev Integration

### HTTP Mode Configuration

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "file-context",
        "transport": {
          "type": "http",
          "url": "http://your-alb-url:3000",
          "headers": {
            "X-MCP-Auth-Token": "your-secure-auth-token"
          }
        }
      }
    ]
  }
}
```

### Local Development (stdio mode)

```json
{
  "experimental": {
    "modelContextProtocolServers": [
      {
        "name": "file-context",
        "transport": {
          "type": "stdio",
          "command": "/path/to/go-mcp-file-context-server",
          "args": ["--root-dir", "/path/to/allowed/directory"]
        }
      }
    ]
  }
}
```

## Testing the Connection

### Using curl

```bash
# Test health endpoint (no auth required)
curl http://your-alb-url:3000/health

# List available tools
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# List allowed directories
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_allowed_directories","arguments":{}},"id":2}'

# Read a file
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"read_context","arguments":{"path":"/data/example.txt"}},"id":3}'
```

## Available Tools

The file-context-server provides these tools:

| Tool | Description |
|------|-------------|
| `list_allowed_directories` | List configured root directories and blocked patterns |
| `list_context_files` | List files in a directory with metadata |
| `read_context` | Read file or directory contents |
| `search_context` | Search for patterns in files |
| `analyze_code` | Analyze code files for metrics |
| `generate_outline` | Generate code outline |
| `cache_stats` | Get cache statistics |
| `get_chunk_count` | Get chunk count for large files |
| `getFiles` | Batch retrieve multiple files |
| `get_folder_structure` | Get folder tree structure |
| `write_file` | Write content to a file |
| `create_directory` | Create a directory |
| `copy_file` | Copy files or directories |
| `move_file` | Move or rename files |
| `delete_file` | Delete files or directories |
| `modify_file` | Find and replace in files |

## Security Best Practices

1. **Use HTTPS**: Always use HTTPS in production
2. **Restrict root directories**: Set `MCP_ROOT_DIR` to limit access
3. **Block sensitive patterns**: Configure `MCP_BLOCKED_PATTERNS`
4. **Rotate tokens**: Implement regular token rotation
5. **Audit access**: Monitor file operations in logs

## Troubleshooting

### 401 Unauthorized
- Verify the `X-MCP-Auth-Token` header matches the server's token

### Access Denied
- Check if the path is within allowed root directories
- Verify the path doesn't match blocked patterns

### File Not Found
- Ensure the file exists at the specified path
- Check path is absolute or relative to root directory
