#!/bin/bash
# Quick upload script untuk Windows PowerShell
# Upload semua file penting ke server Debian

$SERVER = "app@172.20.100.11"
$DEST = "~/apps/forming/"

Write-Host "📤 Uploading files to Debian server..." -ForegroundColor Green

# Upload individual files
Write-Host "📄 Uploading main files..." -ForegroundColor Yellow
scp Dockerfile docker-compose.yml deploy.sh README.md DEPLOYMENT.md $SERVER`:$DEST
scp go.mod go.sum main.go skip_log.go $SERVER`:$DEST

# Upload directories
Write-Host "📂 Uploading views folder..." -ForegroundColor Yellow
scp -r views $SERVER`:$DEST

Write-Host "📂 Uploading public folder..." -ForegroundColor Yellow
scp -r public $SERVER`:$DEST

Write-Host ""
Write-Host "✅ Upload complete!" -ForegroundColor Green
Write-Host ""
Write-Host "🚀 Next steps:" -ForegroundColor Cyan
Write-Host "   ssh $SERVER"
Write-Host "   cd ~/apps/forming"
Write-Host "   chmod +x deploy.sh"
Write-Host "   ./deploy.sh"
Write-Host ""
