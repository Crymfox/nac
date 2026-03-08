# nac - n8n As Code Installer for Windows
# This script downloads the latest nac binary and adds it to your user PATH.

$ErrorActionPreference = "Stop"

$repo = "crymfox/nac"
$releaseUrl = "https://github.com/$repo/releases/latest/download/nac_Windows_x86_64.zip"
$zipFile = "$PSScriptRoot\nac.zip"
$extractDir = "$PSScriptRoot\nac-dist"
$binDir = "$Home\bin"

Write-Host "--- nac Installer for Windows ---" -ForegroundColor Cyan

try {
    # 1. Download
    Write-Host "Downloading latest release..."
    Invoke-WebRequest -Uri $releaseUrl -OutFile $zipFile

    # 2. Extract
    Write-Host "Extracting binary..."
    if (Test-Path $extractDir) { Remove-Item -Path $extractDir -Recurse -Force }
    Expand-Archive -Path $zipFile -DestinationPath $extractDir -Force

    # 3. Setup bin directory
    Write-Host "Setting up $binDir..."
    if (!(Test-Path $binDir)) {
        New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    }

    # 4. Install binary
    Write-Host "Installing nac.exe..."
    Move-Item -Path "$extractDir\nac.exe" -Destination "$binDir\nac.exe" -Force

    # 5. Update PATH for current session
    Write-Host "Updating environment variables..."
    if ($env:Path -notlike "*$binDir*") {
        $env:Path += ";$binDir"
    }

    # 6. Update PATH permanently for the User
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$binDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$binDir", "User")
        Write-Host "Added $binDir to User PATH permanently." -ForegroundColor Green
    }

    # 7. Cleanup
    Write-Host "Cleaning up temporary files..."
    if (Test-Path $zipFile) { Remove-Item -Path $zipFile -Force }
    if (Test-Path $extractDir) { Remove-Item -Path $extractDir -Recurse -Force }

    Write-Host "`nnac has been installed successfully!" -ForegroundColor Green
    Write-Host "You may need to restart your terminal for changes to take effect in other windows."
    Write-Host "Running 'nac version' to verify..." -ForegroundColor Gray

    & "$binDir\nac.exe" version
}
catch {
    Write-Host "`nError: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
