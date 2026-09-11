# Auto deploy for "Testga" button: commit/push (best-effort), build API+UI, copy static, pm2 restart.
# Usage:
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\agent-deploy-test.ps1
# Env (optional):
#   AGENT_APP_DIR  - repo root (default: parent of scripts/)
#   PM2_TEST_NAME  - default ref-ai-test
#   AGENT_TASK_ID  - for commit message

$ErrorActionPreference = "Stop"

$appDir = $env:AGENT_APP_DIR
if (-not $appDir) {
  $appDir = Split-Path -Parent $PSScriptRoot
}
$appDir = (Resolve-Path $appDir).Path
$rootDir = Split-Path -Parent $appDir
$pm2Name = if ($env:PM2_TEST_NAME) { $env:PM2_TEST_NAME } else { "ref-ai-test" }
$taskId = if ($env:AGENT_TASK_ID) { $env:AGENT_TASK_ID } else { "manual" }

Write-Host "== agent-deploy-test =="
Write-Host "appDir=$appDir"
Write-Host "rootDir=$rootDir"
Write-Host "pm2=$pm2Name task=$taskId"

$env:Path = $env:Path + ";C:\Program Files\Git\cmd;C:\Program Files\Go\bin"

Set-Location $appDir

Write-Host "== git status =="
git status -sb

Write-Host "== git add/commit/push (best-effort) =="
try {
  git add -A
  $pending = git status --porcelain
  if ($pending) {
    git -c user.email="agent@local" -c user.name="ref-ai-agent" commit -m "agent task #$taskId auto-deploy"
  } else {
    Write-Host "No changes to commit."
  }
  git push ref-ai-test main 2>&1 | Write-Host
} catch {
  Write-Host "WARN git commit/push: $($_.Exception.Message)"
}

Write-Host "== go build =="
New-Item -ItemType Directory -Force -Path (Join-Path $rootDir "bin") | Out-Null
$exe = Join-Path $rootDir "bin\api_v3.exe"
go build -o $exe .\cmd\api_v3
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Write-Host "== ui build =="
$uiDir = Join-Path $appDir "ui"
Push-Location $uiDir
if (-not (Test-Path "node_modules")) {
  npm ci
}
npm run build
if ($LASTEXITCODE -ne 0) {
  Pop-Location
  throw "npm run build failed"
}
Pop-Location

Write-Host "== copy web/build =="
$srcBuild = Join-Path $appDir "web\build"
if (-not (Test-Path $srcBuild)) {
  $srcBuild = Join-Path $uiDir "dist"
}
$dstBuild = Join-Path $rootDir "web\build"
New-Item -ItemType Directory -Force -Path $dstBuild | Out-Null
Copy-Item -Path (Join-Path $srcBuild "*") -Destination $dstBuild -Recurse -Force

if ($env:AGENT_SKIP_PM2 -eq "1") {
  Write-Host "== skip pm2 (API will restart after status update) =="
} else {
  Write-Host "== pm2 restart $pm2Name =="
  pm2 restart $pm2Name --update-env
  if ($LASTEXITCODE -ne 0) { throw "pm2 restart failed" }
}

Write-Host "OK agent-deploy-test finished"
