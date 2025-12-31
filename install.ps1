# mcp-codewizard installer/updater for Windows
# Usage: irm https://raw.githubusercontent.com/spetr/mcp-codewizard/main/install.ps1 | iex

# Support both piped execution and direct script execution
# For piped execution: set $env:MCP_VERSION and $env:MCP_INSTALL_DIR before running
# For direct execution: use -Version and -InstallDir parameters
param(
    [string]$Version = "",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

# Handle piped execution where param() doesn't work
if (-not $InstallDir) {
    $InstallDir = if ($env:MCP_INSTALL_DIR) { $env:MCP_INSTALL_DIR } else { "$env:LOCALAPPDATA\mcp-codewizard" }
}
if (-not $Version) {
    $Version = $env:MCP_VERSION
}
$ProgressPreference = "SilentlyContinue"

$Repo = "spetr/mcp-codewizard"
$BinaryName = "mcp-codewizard"

function Write-Info($msg) {
    Write-Host "[INFO] " -ForegroundColor Green -NoNewline
    Write-Host $msg
}

function Write-Warn($msg) {
    Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline
    Write-Host $msg
}

function Write-Err($msg) {
    Write-Host "[ERROR] " -ForegroundColor Red -NoNewline
    Write-Host $msg
    exit 1
}

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64" { return "amd64" }
        "Arm64" { return "arm64" }
        default { Write-Err "Unsupported architecture: $arch" }
    }
}

function Get-LatestVersion {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        return $release.tag_name
    } catch {
        Write-Err "Failed to get latest version: $_"
    }
}

function Install-Binary {
    $arch = Get-Architecture
    $version = if ($Version) { $Version } else { Get-LatestVersion }

    if (-not $version) {
        Write-Err "Failed to determine version"
    }

    Write-Info "Installing $BinaryName $version for windows/$arch..."

    $filename = "$BinaryName-$version-windows-$arch.zip"
    $url = "https://github.com/$Repo/releases/download/$version/$filename"

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    # Create temp directory
    $tmpDir = Join-Path $env:TEMP "mcp-codewizard-install"
    if (Test-Path $tmpDir) {
        Remove-Item -Recurse -Force $tmpDir
    }
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        # Download
        Write-Info "Downloading $url..."
        $zipPath = Join-Path $tmpDir $filename
        Invoke-WebRequest -Uri $url -OutFile $zipPath

        # Extract
        Write-Info "Extracting..."
        Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

        # Install
        $binary = Join-Path $tmpDir "$BinaryName.exe"
        if (-not (Test-Path $binary)) {
            Write-Err "Binary not found in archive"
        }

        $destPath = Join-Path $InstallDir "$BinaryName.exe"
        Move-Item -Path $binary -Destination $destPath -Force

        # Add to PATH if not already there
        $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($currentPath -notlike "*$InstallDir*") {
            Write-Info "Adding $InstallDir to PATH..."
            [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
            $env:Path = "$env:Path;$InstallDir"
        }

        # Verify
        Write-Info "Successfully installed $BinaryName to $destPath"
        Write-Info "Restart your terminal or run: `$env:Path = [Environment]::GetEnvironmentVariable('Path', 'User')"

    } finally {
        # Cleanup
        if (Test-Path $tmpDir) {
            Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
        }
    }
}

function Update-Binary {
    $current = ""
    try {
        $current = & "$BinaryName" version 2>$null | Select-Object -First 1
    } catch {}

    $latest = Get-LatestVersion

    if (-not $current) {
        Write-Info "$BinaryName is not installed. Installing..."
        Install-Binary
    } elseif ($current -eq $latest -or "v$current" -eq $latest) {
        Write-Info "Already at latest version: $latest"
    } else {
        Write-Info "Update available: $current -> $latest"
        Install-Binary
    }
}

# Main
Install-Binary
