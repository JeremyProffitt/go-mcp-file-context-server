# ECS Deployment Guide for go-mcp-file-context-server

This guide covers deploying go-mcp-file-context-server as an HTTP service on AWS ECS (Elastic Container Service) using either Fargate or EC2 launch types.

## LLM Usage Notes

When the MCP server is deployed on ECS, LLMs connect via HTTP instead of stdio. Key differences:

| Aspect | Local (stdio) | ECS (HTTP) |
|--------|---------------|------------|
| Connection | Direct process | HTTP to ALB endpoint |
| Authentication | None (local) | `X-MCP-Auth-Token` header required |
| File access | Local filesystem | EFS-mounted paths only |
| Root restriction | `-root-dir` flag | `MCP_ROOT_DIR` env var |

**Important for LLMs:** When working with ECS-deployed servers:
- All file paths are relative to the EFS mount point (typically `/data`)
- Use `list_allowed_directories` to see what paths are accessible
- Authentication errors (401) mean the token is missing or incorrect

---

## Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   MCP Client    │────▶│  Load Balancer  │────▶│   ECS Service   │
│ (Claude Code/   │     │     (ALB)       │     │   (Fargate/EC2) │
│  Continue.dev)  │     └─────────────────┘     └─────────────────┘
└─────────────────┘              │                       │
                                 │                       ▼
                                 │              ┌─────────────────┐
                                 │              │   EFS / EBS     │
                                 │              │   (File Store)  │
                                 │              └─────────────────┘
                                 ▼
                        ┌─────────────────┐
                        │ Secrets Manager │
                        └─────────────────┘
```

## Prerequisites

1. AWS CLI configured with appropriate permissions
2. Docker installed locally for building images
3. An ECR repository created for the image
4. VPC with subnets configured for ECS
5. (Optional) EFS file system for persistent storage

## Quick Start

### 1. Build and Push Docker Image

```bash
# Authenticate to ECR
aws ecr get-login-password --region YOUR_REGION | docker login --username AWS --password-stdin YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com

# Build the image
docker build -t go-mcp-file-context-server .

# Tag for ECR
docker tag go-mcp-file-context-server:latest YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-file-context-server:latest

# Push to ECR
docker push YOUR_ACCOUNT_ID.dkr.ecr.YOUR_REGION.amazonaws.com/go-mcp-file-context-server:latest
```

### 2. Create Secrets in AWS Secrets Manager

```bash
aws secretsmanager create-secret \
    --name mcp/file-context \
    --secret-string '{
        "MCP_AUTH_TOKEN": "your-secure-auth-token"
    }'
```

### 3. Create EFS File System (Optional but Recommended)

```bash
# Create EFS file system
aws efs create-file-system \
    --performance-mode generalPurpose \
    --throughput-mode bursting \
    --tags Key=Name,Value=mcp-file-context-data

# Create mount targets in each subnet
aws efs create-mount-target \
    --file-system-id fs-XXXXXXXX \
    --subnet-id subnet-XXXXXXXX \
    --security-groups sg-XXXXXXXX
```

### 4. Create ECS Resources

```bash
# Create CloudWatch Log Group
aws logs create-log-group --log-group-name /ecs/go-mcp-file-context-server

# Register Task Definition
aws ecs register-task-definition --cli-input-json file://ecs-task-definition.json

# Create ECS Cluster (if not exists)
aws ecs create-cluster --cluster-name mcp-servers

# Create Service
aws ecs create-service \
    --cluster mcp-servers \
    --service-name go-mcp-file-context-server \
    --task-definition go-mcp-file-context-server \
    --desired-count 1 \
    --launch-type FARGATE \
    --platform-version 1.4.0 \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx],securityGroups=[sg-xxx],assignPublicIp=ENABLED}"
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `MCP_AUTH_TOKEN` | No | Token for HTTP authentication |
| `MCP_ROOT_DIR` | No | Root directories to restrict access (comma-separated) |
| `MCP_BLOCKED_PATTERNS` | No | Patterns to block (default: .aws/*,.env,.mcp_env) |
| `MCP_LOG_LEVEL` | No | Log level (default: info) |
| `MCP_LOG_DIR` | No | Directory for log files |

### Authentication

When `MCP_AUTH_TOKEN` is set, all HTTP requests must include the `X-MCP-Auth-Token` header.

```bash
curl -X POST http://your-alb-url:3000/ \
    -H "Content-Type: application/json" \
    -H "X-MCP-Auth-Token: your-secure-auth-token" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

## Storage Options

### Option 1: EFS (Recommended for Persistent Storage)

The task definition includes EFS volume configuration. Update the `fileSystemId` with your EFS ID.

Benefits:
- Persistent storage across task restarts
- Shared storage between multiple tasks
- Automatic backup capability

### Option 2: Ephemeral Storage

For temporary file operations, use the default container storage:
- Remove the `volumes` and `mountPoints` from task definition
- Set `--root-dir` to a container path like `/tmp/data`

### Option 3: S3 (via S3FS)

For S3-based storage, add s3fs-fuse to the container and mount an S3 bucket.

## Security Considerations

1. **Root Directory Restriction**: Always set `MCP_ROOT_DIR` to limit file access
2. **Blocked Patterns**: Configure `MCP_BLOCKED_PATTERNS` to block sensitive files
3. **Authentication**: Enable `MCP_AUTH_TOKEN` for production
4. **Private Subnets**: Deploy in private subnets when possible
5. **EFS Encryption**: Enable encryption for EFS volumes
6. **IAM Permissions**: Minimal task role permissions

## Monitoring

### CloudWatch Logs

Logs are sent to CloudWatch Logs at `/ecs/go-mcp-file-context-server`.

### Health Checks

The service exposes a `/health` endpoint that returns:
```json
{"status": "healthy", "server": "file-context-server"}
```

## Troubleshooting

### Common Issues

1. **EFS mount failures**: Check security groups allow NFS traffic (port 2049)
2. **Permission denied**: Ensure the mcpuser has access to mounted volumes
3. **Path outside root**: Check `MCP_ROOT_DIR` configuration

See [INTEGRATION.md](./INTEGRATION.md) for configuring Claude Code and Continue.dev.
