# Verifies a deployed app URL: Traefik router, dynamic YAML, and HTTP via Host header.
# Usage:
#   .\scripts\verify-deploy-route.ps1 -HostHeader "web-xxx.app-yyy.nebula.localhost"
#   .\scripts\verify-deploy-route.ps1 -ServiceUrl "http://web-xxx.app-yyy.nebula.localhost"

param(
  [string]$HostHeader = "",
  [string]$ServiceUrl = "",
  [string]$TraefikAPI = "http://127.0.0.1:8080",
  [string]$DynamicDir = "deployments/traefik/dynamic"
)

$ErrorActionPreference = "Stop"

if ($ServiceUrl) {
  $u = [Uri]$ServiceUrl
  $HostHeader = $u.Host
}

if (-not $HostHeader) {
  throw "Provide -HostHeader or -ServiceUrl"
}

Write-Host "== Traefik routers ==" -ForegroundColor Cyan
$routers = Invoke-RestMethod "$TraefikAPI/api/http/routers"
$match = $routers | Where-Object { $_.rule -like "*$HostHeader*" }
if ($match) {
  $match | ForEach-Object { Write-Host "OK router: $($_.name) rule=$($_.rule)" -ForegroundColor Green }
} else {
  Write-Host "WARN: no router rule contains Host $HostHeader" -ForegroundColor Yellow
}

Write-Host "== dynamic YAML ==" -ForegroundColor Cyan
$yamlHits = Get-ChildItem -Path $DynamicDir -Filter "nebula-*.yml" -ErrorAction SilentlyContinue |
  Where-Object { (Get-Content $_.FullName -Raw) -match [regex]::Escape($HostHeader) }
if ($yamlHits) {
  $yamlHits | ForEach-Object { Write-Host "OK file: $($_.Name)" -ForegroundColor Green }
} else {
  Write-Host "WARN: no nebula-*.yml under $DynamicDir mentions $HostHeader" -ForegroundColor Yellow
}

Write-Host "== HTTP probe ==" -ForegroundColor Cyan
try {
  $r = Invoke-WebRequest -Uri "http://127.0.0.1/" -Headers @{ Host = $HostHeader } -UseBasicParsing
  Write-Host "OK status $($r.StatusCode) for Host $HostHeader" -ForegroundColor Green
} catch {
  if ($_.Exception.Response.StatusCode.value__ -eq 404) {
    Write-Host "FAIL 404 — try: docker compose restart traefik" -ForegroundColor Red
    exit 1
  }
  throw
}

Write-Host "Done." -ForegroundColor Green
