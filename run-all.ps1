<#
.SYNOPSIS
  Starts the whole Spotto stack (Postgres check, API, vendor webapp, admin webapp)
  each in its own window, then reports readiness.

.PARAMETER SkipSeed
  Skip `go run ./cmd/seed` - keeps whatever is already in the database.

.PARAMETER App
  Also launch the Flutter traveller app (`flutter run`) in a 5th window.

.EXAMPLE
  .\run-all.ps1
  .\run-all.ps1 -SkipSeed
  .\run-all.ps1 -App
#>
param(
    [switch]$SkipSeed,
    [switch]$App
)

$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot

function Write-Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg)   { Write-Host "    $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "    $msg" -ForegroundColor Yellow }

function Wait-Url($url, $timeoutSec) {
    $deadline = (Get-Date).AddSeconds($timeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $res = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 3
            if ($res.StatusCode -eq 200) { return $true }
        } catch { }
        Start-Sleep -Seconds 1
    }
    return $false
}

function Start-TitledWindow($title, $workDir, $commandLine) {
    $titleSet = '$host.UI.RawUI.WindowTitle' + " = '" + $title + "'"
    $full = $titleSet + '; cd ' + "'" + $workDir + "'" + '; ' + $commandLine
    Start-Process powershell -ArgumentList '-NoExit', '-Command', $full -WindowStyle Normal | Out-Null
}

# --- 1. Docker check ---------------------------------------------------
Write-Step "Checking Docker..."
$dockerUp = $true
try { docker info *> $null } catch { $dockerUp = $false }
if (-not $dockerUp) {
    Write-Warn "Docker not responding - starting Docker Desktop..."
    Start-Process "C:\Program Files\Docker\Docker\Docker Desktop.exe"
    $deadline = (Get-Date).AddSeconds(90)
    while ((Get-Date) -lt $deadline) {
        try { docker info *> $null; $dockerUp = $true; break } catch { Start-Sleep -Seconds 3 }
    }
    if (-not $dockerUp) {
        Write-Host "Docker did not come up in time. Start it manually and re-run." -ForegroundColor Red
        exit 1
    }
}
Write-Ok "Docker is up."

# --- 2. Database check --------------------------------------------------
Write-Step "Checking spotto database..."
$exists = docker exec shared-postgres psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='spotto'" 2>$null
if ($exists -match '1') {
    Write-Ok "Database 'spotto' already exists."
} else {
    Write-Warn "Creating database 'spotto'..."
    docker exec shared-postgres psql -U postgres -c "CREATE DATABASE spotto" | Out-Null
    Write-Ok "Created."
}

# --- 3. API window -------------------------------------------------------
Write-Step "Starting API (:8080)..."
$seedCmd = if ($SkipSeed) { '' } else { 'go run ./cmd/seed; ' }
Start-TitledWindow -title 'Spotto API' -workDir "$root\server" -commandLine ($seedCmd + 'go run ./cmd/api')

# --- 4. Wait for API health ----------------------------------------------
Write-Step "Waiting for API health check..."
if (Wait-Url "http://localhost:8080/health" 30) {
    Write-Ok "API is up at http://localhost:8080"
} else {
    Write-Warn "API did not respond within 30s - check its window for errors."
}

# --- 5. Vendor window -----------------------------------------------------
Write-Step "Starting vendor webapp (:3000)..."
$installCmd = if (Test-Path "$root\web\node_modules") { '' } else { 'npm install; ' }
Start-TitledWindow -title 'Spotto Vendor 3000' -workDir "$root\web" -commandLine ($installCmd + 'npm run dev:vendor')

# --- 6. Admin window -------------------------------------------------------
Write-Step "Starting admin webapp (:3001)..."
Start-TitledWindow -title 'Spotto Admin 3001' -workDir "$root\web" -commandLine 'npm run dev:admin'

# --- 7. Poll readiness ------------------------------------------------------
Write-Step "Waiting for vendor + admin webapps..."
$vendorUp = Wait-Url "http://localhost:3000/login" 60
$adminUp  = Wait-Url "http://localhost:3001/login" 60
if ($vendorUp) { Write-Ok "Vendor portal is up at http://localhost:3000" }
else { Write-Warn "Vendor portal did not respond within 60s - check its window." }
if ($adminUp) { Write-Ok "Admin center is up at http://localhost:3001" }
else { Write-Warn "Admin center did not respond within 60s - check its window." }

# --- 8. Optional Flutter app -------------------------------------------------
if ($App) {
    Write-Step "Starting traveller app (flutter run)..."
    Start-TitledWindow -title 'Spotto Traveller App' -workDir "$root\app" -commandLine 'flutter run'
}

# --- 9. Summary ---------------------------------------------------------------
Write-Host ""
Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host " Spotto stack"
Write-Host "=========================================================" -ForegroundColor Cyan
Write-Host " API           http://localhost:8080/health"
Write-Host " Vendor portal http://localhost:3000  (vendor@driftwood.dev / spotto123)"
Write-Host " Admin center  http://localhost:3001  (admin@spotto.dev / spotto123)"
Write-Host " Traveller app lena@spotto.dev / spotto123"
if (-not $App) {
    Write-Host ""
    Write-Host " Traveller app wasn't started. Run it yourself with:"
    Write-Host "   cd app; flutter run"
}
Write-Host "=========================================================" -ForegroundColor Cyan
