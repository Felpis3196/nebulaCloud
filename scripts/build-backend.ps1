# Build backend binaries (Windows equivalent of `make build`)
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location (Join-Path $root "backend")
New-Item -ItemType Directory -Force -Path bin | Out-Null
go build -o bin/api.exe ./cmd/api
go build -o bin/build-worker.exe ./cmd/build-worker
go build -o bin/runtime-agent.exe ./cmd/runtime-agent
Write-Host "Built: backend/bin/api.exe, build-worker.exe, runtime-agent.exe"
