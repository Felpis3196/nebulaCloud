# NebulaCloud E2E smoke — login, org, project, repo, service, deploy enqueue
# Requires: API at http://localhost:8081, full stack for deploy completion (build-worker + runtime-agent)
#
# Usage:
#   .\scripts\e2e-smoke.ps1
#   $env:NEBULA_TEST_DSN = "postgres://nebula:nebula@localhost:5432/nebula?sslmode=disable"

param(
  [string]$ApiBase = "http://localhost:8081",
  [string]$RepoUrl = "https://github.com/docker/welcome-to-docker",
  [switch]$WaitForDeploy,
  [int]$DeployTimeoutSec = 600
)

$ErrorActionPreference = "Stop"

function Invoke-NebulaJson {
  param(
    [string]$Method,
    [string]$Path,
    [object]$Body = $null,
    [string]$Token = $null
  )
  $headers = @{ Accept = "application/json" }
  if ($Body) { $headers["Content-Type"] = "application/json" }
  if ($Token) { $headers["Authorization"] = "Bearer $Token" }
  $uri = "$ApiBase$Path"
  $params = @{ Method = $Method; Headers = $headers }
  if ($Body) { $params["Body"] = ($Body | ConvertTo-Json -Compress) }
  $r = Invoke-WebRequest @params -Uri $uri -UseBasicParsing
  if ($r.Content) { return ($r.Content | ConvertFrom-Json) }
  return $null
}

Write-Host "== healthz ==" -ForegroundColor Cyan
$h = Invoke-WebRequest "$ApiBase/healthz" -UseBasicParsing
if ($h.StatusCode -ne 200) { throw "API not healthy" }

$email = "smoke-$(Get-Random)@nebula.test"
$pass = "valid-password-12"

Write-Host "== register + login ==" -ForegroundColor Cyan
try {
  Invoke-NebulaJson POST "/api/v1/auth/register" @{ email = $email; password = $pass; display_name = "Smoke" }
} catch {
  Write-Host "register: $($_.Exception.Message)"
}
$login = Invoke-NebulaJson POST "/api/v1/auth/login" @{ email = $email; password = $pass }
$token = $login.data.access_token
if (-not $token) { throw "no access token" }

Write-Host "== organization ==" -ForegroundColor Cyan
$slug = "smoke-" + [guid]::NewGuid().ToString("N").Substring(0, 8)
$org = Invoke-NebulaJson POST "/api/v1/organizations" @{ slug = $slug; name = "Smoke Org" } -Token $token
$orgId = $org.data.id

Write-Host "== project ==" -ForegroundColor Cyan
$proj = Invoke-NebulaJson POST "/api/v1/organizations/$orgId/projects" @{
  slug = "app"; name = "Smoke App"; default_branch = "main"
} -Token $token
$projId = $proj.data.id

Write-Host "== membership denied (second user) ==" -ForegroundColor Cyan
$email2 = "smoke-b-$(Get-Random)@nebula.test"
Invoke-NebulaJson POST "/api/v1/auth/register" @{ email = $email2; password = $pass } | Out-Null
$login2 = Invoke-NebulaJson POST "/api/v1/auth/login" @{ email = $email2; password = $pass }
$token2 = $login2.data.access_token
try {
  Invoke-NebulaJson POST "/api/v1/organizations/$orgId/projects" @{ slug = "hack"; name = "Hack" } -Token $token2
  throw "expected membership failure for user B"
} catch {
  if ($_.Exception.Response.StatusCode.value__ -ne 404) {
    Write-Host "membership check: $($_.Exception.Message) (expected 404)"
  } else {
    Write-Host "OK: user B cannot create project in A's org"
  }
}

Write-Host "== connect repo ==" -ForegroundColor Cyan
Invoke-NebulaJson PATCH "/api/v1/projects/$projId" @{
  repo_url = $RepoUrl
  default_branch = "master"
} -Token $token | Out-Null

Write-Host "== service ==" -ForegroundColor Cyan
$svc = Invoke-NebulaJson POST "/api/v1/projects/$projId/services" @{
  slug = "web"; name = "Web"; type = "web"
} -Token $token
$svcId = $svc.data.id

Write-Host "== deploy enqueue ==" -ForegroundColor Cyan
$dep = Invoke-NebulaJson POST "/api/v1/services/$svcId/deployments" @{} -Token $token
$depId = $dep.data.id
Write-Host "deployment id: $depId status: $($dep.data.status)"

if ($WaitForDeploy) {
  Write-Host "== wait for deployment ==" -ForegroundColor Cyan
  $deadline = (Get-Date).AddSeconds($DeployTimeoutSec)
  $final = $null
  while ((Get-Date) -lt $deadline) {
    $final = Invoke-NebulaJson GET "/api/v1/deployments/$depId" $null -Token $token
    $d = $final.data
    if (-not $d) { $d = $final }
    $st = $d.status
    Write-Host "  status: $st"
    if ($st -in @("running", "failed")) { break }
    Start-Sleep -Seconds 5
  }
  if ($d.status -ne "running") {
    throw "deployment did not reach running: $($d.status) $($d.error_message)"
  }
  if ($d.route_host) {
    Write-Host "== verify public URL ==" -ForegroundColor Cyan
    & "$PSScriptRoot/verify-deploy-route.ps1" -HostHeader $d.route_host
  } else {
    Write-Host "WARN: deployment has no route_host in API response" -ForegroundColor Yellow
  }
}

Write-Host ""
Write-Host "Smoke API flow completed." -ForegroundColor Green
Write-Host "For running containers, ensure full stack:"
Write-Host "  docker compose up -d --build"
Write-Host "  docker compose ps   # build-worker + runtime-agent healthy"
Write-Host "Optional end-to-end: .\scripts\e2e-smoke.ps1 -WaitForDeploy"
Write-Host "Then watch: docker compose logs -f build-worker runtime-agent"
