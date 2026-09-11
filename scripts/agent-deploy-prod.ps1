# Promote test artifacts → prod (no rebuild). Used by "Prodga" button.
# Env:
#   REF_AI_TEST_DIR  default D:\ref-main\ref-ai\ref-ai-test
#   REF_AI_PROD_DIR  default D:\ref-main\ref-ai\ref-ai-prod
#   PM2_PROD_NAME    default ref-ai-prod
#   AGENT_SKIP_PM2=1 — skip pm2 (API restarts after DB status)
#   AGENT_TASK_ID

$ErrorActionPreference = "Stop"

$testDir = if ($env:REF_AI_TEST_DIR) { $env:REF_AI_TEST_DIR } else { "D:\ref-main\ref-ai\ref-ai-test" }
$prodDir = if ($env:REF_AI_PROD_DIR) { $env:REF_AI_PROD_DIR } else { "D:\ref-main\ref-ai\ref-ai-prod" }
$pm2Name = if ($env:PM2_PROD_NAME) { $env:PM2_PROD_NAME } else { "ref-ai-prod" }
$taskId = if ($env:AGENT_TASK_ID) { $env:AGENT_TASK_ID } else { "manual" }
$binary = if ($env:APP_BINARY) { $env:APP_BINARY } else { "api_v3.exe" }

Write-Host "== agent-deploy-prod =="
Write-Host "task=$taskId"
Write-Host "testDir=$testDir"
Write-Host "prodDir=$prodDir"
Write-Host "pm2=$pm2Name"

if (-not (Test-Path $testDir)) { throw "REF_AI_TEST_DIR topilmadi: $testDir" }

$srcExe = Join-Path $testDir "bin\$binary"
$srcWeb = Join-Path $testDir "web\build"
if (-not (Test-Path $srcExe)) { throw "Test binary yo'q: $srcExe" }
if (-not (Test-Path $srcWeb)) { throw "Test web/build yo'q: $srcWeb" }

New-Item -ItemType Directory -Force -Path (Join-Path $prodDir "bin") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $prodDir "web\build") | Out-Null

Write-Host "== copy binary =="
Copy-Item -Path $srcExe -Destination (Join-Path $prodDir "bin\$binary") -Force

Write-Host "== copy web/build =="
Copy-Item -Path (Join-Path $srcWeb "*") -Destination (Join-Path $prodDir "web\build") -Recurse -Force

# Seed ecosystem if missing (does not overwrite .env)
$ecoSrc = Join-Path $PSScriptRoot "ecosystem.ref-ai-prod.cjs"
$ecoDst = Join-Path $prodDir "ecosystem.config.cjs"
if ((Test-Path $ecoSrc) -and -not (Test-Path $ecoDst)) {
  Write-Host "== seed ecosystem.config.cjs =="
  Copy-Item $ecoSrc $ecoDst -Force
}

if ($env:AGENT_SKIP_PM2 -eq "1") {
  Write-Host "== skip pm2 (API will restart prod after status update) =="
} else {
  Write-Host "== pm2 restart/start $pm2Name =="
  pm2 describe $pm2Name 2>$null | Out-Null
  if ($LASTEXITCODE -eq 0) {
    pm2 restart $pm2Name --update-env
  } else {
    if (-not (Test-Path $ecoDst)) { throw "PM2 app yo'q va ecosystem.config.cjs topilmadi: $ecoDst" }
    Push-Location $prodDir
    pm2 start $ecoDst
    Pop-Location
  }
  if ($LASTEXITCODE -ne 0) { throw "pm2 restart/start failed" }
}

Write-Host "OK agent-deploy-prod finished"
