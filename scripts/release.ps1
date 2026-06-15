param(
    [string]$Version = "1.0.0",
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

$platforms = @(
    @{OS="linux"; Arch="amd64"},
    @{OS="linux"; Arch="arm64"},
    @{OS="darwin"; Arch="amd64"},
    @{OS="darwin"; Arch="arm64"},
    @{OS="windows"; Arch="amd64"}
)

Write-Host "Building NetCheck v$Version" -ForegroundColor Green

New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

foreach ($p in $platforms) {
    $ext = if ($p.OS -eq "windows") { ".exe" } else { "" }
    $output = Join-Path $OutputDir "netcheck-$($p.OS)-$($p.Arch)$ext"
    
    Write-Host "Building for $($p.OS)/$($p.Arch)..."
    
    $env:GOOS = $p.OS
    $env:GOARCH = $p.Arch
    $env:CGO_ENABLED = "0"
    
    & go build -ldflags="-s -w -X main.version=$Version" -o $output ./cmd/netcheck
    
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed for $($p.OS)/$($p.Arch)"
        exit 1
    }
}

Write-Host "Build complete. Artifacts in $OutputDir/" -ForegroundColor Green
Get-ChildItem -Path $OutputDir
