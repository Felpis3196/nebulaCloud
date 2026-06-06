# Run backend compile + vet + unit tests (no Docker required)
$ErrorActionPreference = "Stop"
$env:PATH = "C:\Program Files\Go\bin;" + $env:PATH
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location (Join-Path $root "backend")

Write-Host "== go mod tidy =="
go mod tidy

Write-Host "== go build =="
go build ./...

Write-Host "== go vet =="
go vet ./...

Write-Host "== go test =="
go test -count=1 ./...

Write-Host "OK: backend build + tests passed"
