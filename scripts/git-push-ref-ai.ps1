# One-shot: stage app changes, commit, push to ref-ai-test.
# Usage (from repo root):
#   powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\git-push-ref-ai.ps1
# Optional message:
#   .\scripts\git-push-ref-ai.ps1 -Message "my message"

param(
  [string]$Message = "Update agent UI/API and migrations.",
  [string]$Remote = "ref-ai-test",
  [string]$Branch = "main"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

git add -A
# keep secrets / local scaffold out (also covered by .gitignore)
git reset HEAD -- .env .env.* ref-ai/ 2>$null

$status = git status --porcelain
if (-not $status) {
  Write-Host "No changes to commit. Pushing branch anyway..."
  git push $Remote $Branch
  exit 0
}

git commit -m $Message
if ($LASTEXITCODE -ne 0) {
  Write-Error "Commit failed"
  exit 1
}

git push $Remote $Branch
if ($LASTEXITCODE -ne 0) {
  Write-Error "Push failed"
  exit 1
}

Write-Host "Done: pushed to $Remote/$Branch"
git log -1 --oneline
git status -sb
