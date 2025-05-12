# Windows build script for FileZap
param (
    [string]$arch = "amd64"
)

# Setup MinGW path
$mingwPaths = @(
    "C:\mingw64\bin",
    "C:\msys64\mingw64\bin",
    "C:\Program Files\mingw-w64_64-8.1.0-posix-seh-rt_v6-rev0\mingw64\bin"
)

$mingwFound = $false
foreach ($path in $mingwPaths) {
    if (Test-Path $path) {
        $env:Path += ";$path"
        Write-Host "Added $path to PATH"
        Write-Host "Testing GCC..."
        gcc --version
        $mingwFound = $true
        break
    }
}

if (-not $mingwFound) {
    Write-Error "MinGW-w64 not found. Please install it and add to PATH"
    exit 1
}

# Set required environment variables
$env:CGO_ENABLED = "1"
$env:FYNE_RENDER = "software"
$env:GOOS = "windows"
$env:GOARCH = $arch

# Build each component
$components = @(
    @{Name="Client"; Main="cmd/client/main.go"; Windows="cmd/client/main_windows.go"; Flags="-tags 'no_native_menus' -ldflags='-H windowsgui'"},
    @{Name="Divider"; Main="cmd/divider/main.go"; Windows=$null; Flags=""},
    @{Name="Network Core"; Main="cmd/networkcore/main.go"; Windows=$null; Flags=""},
    @{Name="Reconstructor"; Main="cmd/reconstructor/main.go"; Windows=$null; Flags=""},
    @{Name="Validator Server"; Main="cmd/validatorserver/main.go"; Windows=$null; Flags=""}
)

foreach ($component in $components) {
    Write-Host "`nBuilding $($component.Name)..."
    Set-Location -Path $component.Name
    
    # Create bin directory if it doesn't exist
    if (-not (Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin"
    }

    # Construct build command
    $output = "bin/$($component.Name.ToLower().Replace(' ', '')).exe"
    $buildCmd = "go build -v -o $output"
    
    if ($component.Flags) {
        $buildCmd += " $($component.Flags)"
    }

    if ($component.Windows) {
        $buildCmd += " $($component.Main) $($component.Windows)"
    } else {
        $buildCmd += " $($component.Main)"
    }

    # Execute build
    Write-Host "Executing: $buildCmd"
    Invoke-Expression $buildCmd

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed for $($component.Name)"
        exit 1
    }

    Set-Location ..
}

Write-Host "`nAll components built successfully!"
