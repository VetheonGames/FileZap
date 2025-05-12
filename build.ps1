# Main build script for FileZap

# Function to show menu and get selection
function Show-Menu {
    param (
        [string]$Title = 'Build Options',
        [array]$Options
    )
    
    Write-Host "`n$Title`n"
    for ($i = 0; $i -lt $Options.Count; $i++) {
        Write-Host "$($i+1). $($Options[$i])"
    }
    Write-Host "`nQ. Quit"
    
    while ($true) {
        $selection = Read-Host "`nSelect an option"
        if ($selection -eq 'Q' -or $selection -eq 'q') { 
            exit 
        }
        
        $index = [int]$selection - 1
        if ($index -ge 0 -and $index -lt $Options.Count) {
            return $index
        }
        
        Write-Host "Invalid selection. Please try again."
    }
}

# Determine current OS
$isWindows = $env:OS -like "*Windows*"
$isMacOS = $IsMacOS
$isLinux = $IsLinux

# Set up OS and architecture options
$osOptions = @(
    "Windows",
    "Linux",
    "macOS"
)

$archOptions = @(
    "amd64 (x64)",
    "386 (x86)",
    "arm64"
)

# Show OS selection menu
Write-Host "FileZap Build System"
Write-Host "==================="

$osIndex = Show-Menu -Title "Select Target Operating System" -Options $osOptions
$archIndex = Show-Menu -Title "Select Target Architecture" -Options $archOptions

# Map selection to actual values
$targetOS = $osOptions[$osIndex].ToLower()
$targetArch = switch ($archIndex) {
    0 { "amd64" }
    1 { "386" }
    2 { "arm64" }
}

# Execute appropriate build script
Write-Host "`nBuilding for $targetOS/$targetArch..."

if ($isWindows) {
    # On Windows, we can directly execute the PowerShell scripts
    switch ($targetOS) {
        "windows" {
            & "$PSScriptRoot\scripts\build_windows.ps1" -arch $targetArch
        }
        "linux" {
            # Check if WSL is available
            $wslCheck = wsl --list
            if ($LASTEXITCODE -eq 0) {
                Write-Host "Building using WSL..."
                wsl bash ./scripts/build_linux.sh $targetArch
            } else {
                Write-Error "WSL is not available. Please install WSL to build for Linux on Windows."
                exit 1
            }
        }
        "macos" {
            Write-Error "Building for macOS is not supported on Windows."
            exit 1
        }
    }
} elseif ($isMacOS) {
    # On macOS, execute bash scripts
    if ($targetOS -eq "windows") {
        Write-Error "Building for Windows is not supported on macOS."
        exit 1
    }
    
    $script = if ($targetOS -eq "linux") { "./scripts/build_linux.sh" } else { "./scripts/build_mac.sh" }
    bash $script $targetArch
} elseif ($isLinux) {
    # On Linux, execute bash scripts
    if ($targetOS -eq "windows" -or $targetOS -eq "macos") {
        Write-Error "Building for $targetOS is not supported on Linux."
        exit 1
    }
    
    bash ./scripts/build_linux.sh $targetArch
} else {
    Write-Error "Unsupported operating system"
    exit 1
}

# Make scripts executable on Unix-like systems
if (-not $isWindows) {
    chmod +x ./scripts/build_linux.sh
    chmod +x ./scripts/build_mac.sh
}

Write-Host "`nBuild process completed!"
