# Auto deploy for "Testga": build API+UI, copy static. Git is OFF by default
# (PM2 often runs as SYSTEM -> dubious ownership / hung credential prompts).
#
# Env:
#   AGENT_APP_DIR, PM2_TEST_NAME, AGENT_TASK_ID
#   AGENT_SKIP_PM2=1  - API restarts pm2 after DB update
#   AGENT_DO_GIT=1    - optional best-effort commit/push (not recommended under SYSTEM)
#
# Binary is written to api_v3.exe.new so the running test process is not locked.
# Go (or this script if AGENT_SKIP_PM2 is unset) stops PM2, swaps, then restarts
# WITHOUT --update-env so PORT/CONN_STRING stay from the app .env.

$ErrorActionPreference = "Stop"

$appDir = $env:AGENT_APP_DIR
if (-not $appDir) {
  $appDir = Split-Path -Parent $PSScriptRoot
}
$appDir = (Resolve-Path $appDir).Path
$rootDir = Split-Path -Parent $appDir
$pm2Name = if ($env:PM2_TEST_NAME) { $env:PM2_TEST_NAME } else { "ref-ai-test" }
$taskId = if ($env:AGENT_TASK_ID) { $env:AGENT_TASK_ID } else { "manual" }
$binary = if ($env:APP_BINARY) { $env:APP_BINARY } else { "api_v3.exe" }

# Never wait for git credentials in non-interactive PM2 sessions
$env:GIT_TERMINAL_PROMPT = "0"
$env:GCM_INTERACTIVE = "never"

Write-Host "== agent-deploy-test =="
Write-Host "appDir=$appDir"
Write-Host "rootDir=$rootDir"
Write-Host "pm2=$pm2Name task=$taskId"

$env:Path = $env:Path + ";C:\Program Files\Git\cmd;C:\Program Files\Go\bin;C:\Program Files\nodejs"

Set-Location $appDir

if ($env:AGENT_DO_GIT -eq "1") {
  Write-Host "== git add/commit/push (best-effort, 60s cap) =="
  try {
    $gitJob = Start-Job -ScriptBlock {
      param($dir, $tid)
      Set-Location $dir
      $env:GIT_TERMINAL_PROMPT = "0"
      git -c safe.directory=* status -sb
      git -c safe.directory=* add -A
      $pending = git -c safe.directory=* status --porcelain
      if ($pending) {
        git -c safe.directory=* -c user.email="agent@local" -c user.name="ref-ai-agent" commit -m "agent task #$tid auto-deploy"
      }
      git -c safe.directory=* push ref-ai-test main 2>&1
    } -ArgumentList $appDir, $taskId

    $finished = Wait-Job $gitJob -Timeout 60
    if (-not $finished) {
      Stop-Job $gitJob -Force
      Remove-Job $gitJob -Force
      Write-Host "WARN git timed out after 60s - skipping"
    } else {
      Receive-Job $gitJob | ForEach-Object { Write-Host $_ }
      Remove-Job $gitJob -Force
    }
  } catch {
    Write-Host "WARN git: $($_.Exception.Message)"
  }
} else {
  Write-Host "== skip git (set AGENT_DO_GIT=1 to enable) =="
}

Write-Host "== go build =="
$binDir = Join-Path $rootDir "bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$exe = Join-Path $binDir $binary
$exeNew = "$exe.new"
# Write to .new so the running api_v3.exe is not locked (Windows).
# -buildvcs=false: avoid exit 128 when git ownership blocks VCS stamping (PM2/SYSTEM)
go build -buildvcs=false -o $exeNew .\cmd\api_v3
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Write-Host "== ui build =="
$uiDir = Join-Path $appDir "ui"
Push-Location $uiDir
if (-not (Test-Path "node_modules")) {
  npm ci
  if ($LASTEXITCODE -ne 0) {
    Pop-Location
    throw "npm ci failed"
  }
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
if (-not (Test-Path $srcBuild)) {
  throw "UI build output topilmadi: $srcBuild"
}
$dstBuild = Join-Path $rootDir "web\build"
New-Item -ItemType Directory -Force -Path $dstBuild | Out-Null
Copy-Item -Path (Join-Path $srcBuild "*") -Destination $dstBuild -Recurse -Force

if ($env:AGENT_SKIP_PM2 -eq "1") {
  Write-Host "== skip pm2 swap/restart (API will apply $binary.new then restart) =="
} else {
  Write-Host "== pm2 stop / swap binary / restart $pm2Name =="
  pm2 stop $pm2Name 2>$null | Out-Null
  if (Test-Path $exeNew) {
    Move-Item -Path $exeNew -Destination $exe -Force
  }
  # Do NOT use --update-env: caller env must not override app .env (PORT/CONN_STRING).
  pm2 restart $pm2Name
  if ($LASTEXITCODE -ne 0) { throw "pm2 restart failed" }
}

Write-Host "OK agent-deploy-test finished"
