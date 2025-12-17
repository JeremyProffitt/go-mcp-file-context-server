# Build script for go-mcp-file-context-server (PowerShell)
# Creates binaries for all supported platforms

$ErrorActionPreference = "Stop"

$APP_NAME = "go-mcp-file-context-server"
$OUTPUT_DIR = "bin"

# Extract version from main.go
$VERSION = (Select-String -Path "main.go" -Pattern 'Version\s*=\s*"([^"]+)"' | Select-Object -First 1).Matches.Groups[1].Value

Write-Host "Building $APP_NAME v$VERSION"
Write-Host "================================"

# Clean and create output directory
if (Test-Path $OUTPUT_DIR) {
    Remove-Item -Recurse -Force $OUTPUT_DIR
}
New-Item -ItemType Directory -Path $OUTPUT_DIR | Out-Null

function Build-Binary {
    param (
        [string]$OS,
        [string]$Arch,
        [string]$Output
    )

    Write-Host "Building for $OS/$Arch..."

    $env:GOOS = $OS
    $env:GOARCH = $Arch

    go build -ldflags="-s -w" -o "$OUTPUT_DIR/$Output" .

    if ($LASTEXITCODE -eq 0) {
        Write-Host "  -> $OUTPUT_DIR/$Output"
    } else {
        Write-Host "  -> FAILED"
        exit 1
    }
}

# macOS builds
Write-Host ""
Write-Host "Building macOS binaries..."
Build-Binary -OS "darwin" -Arch "arm64" -Output "${APP_NAME}-darwin-arm64"
Build-Binary -OS "darwin" -Arch "amd64" -Output "${APP_NAME}-darwin-amd64"
Write-Host "Note: Universal binary must be created on macOS using lipo"

# Linux builds
Write-Host ""
Write-Host "Building Linux binaries..."
Build-Binary -OS "linux" -Arch "amd64" -Output "${APP_NAME}-linux-amd64"
Build-Binary -OS "linux" -Arch "arm64" -Output "${APP_NAME}-linux-arm64"

# Windows builds
Write-Host ""
Write-Host "Building Windows binaries..."
Build-Binary -OS "windows" -Arch "amd64" -Output "${APP_NAME}-windows-amd64.exe"

# Generate checksums
Write-Host ""
Write-Host "Generating checksums..."
Push-Location $OUTPUT_DIR
Get-ChildItem -File | ForEach-Object {
    $hash = (Get-FileHash -Path $_.Name -Algorithm SHA256).Hash.ToLower()
    "$hash  $($_.Name)"
} | Out-File -FilePath "checksums.txt" -Encoding UTF8
Pop-Location

Write-Host ""
Write-Host "================================"
Write-Host "Build complete!"
Write-Host ""
Write-Host "Output directory: $OUTPUT_DIR/"
Get-ChildItem $OUTPUT_DIR | Format-Table Name, Length

# Reset environment
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
