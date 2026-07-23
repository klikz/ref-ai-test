$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

if (-not (Test-Path "node_modules\concurrently")) {
    Write-Host "Installing dev dependencies..."
    npm install
}

npm run dev
