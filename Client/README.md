# FileZap Client

A cross-platform GUI client for the FileZap file sharing system. This client allows you to split files into encrypted chunks, join them back together, and interact with the FileZap network.

## Prerequisites

### Windows
1. Install [MSYS2](https://www.msys2.org/)
2. After installing MSYS2, open MSYS2 terminal and run:
   ```bash
   pacman -S mingw-w64-x86_64-gcc
   ```
3. Add `C:\msys64\mingw64\bin` to your system PATH

### Linux
```bash
# Ubuntu/Debian
sudo apt-get install gcc libgl1-mesa-dev xorg-dev

# Fedora
sudo dnf install gcc libXcursor-devel libXrandr-devel mesa-libGL-devel libXinerama-devel libXi-devel

# Arch
sudo pacman -S gcc libgl xorg-server
```

### macOS
```bash
brew install gcc
```

## Building

### Windows
Run the build script in PowerShell:
```powershell
./build.ps1
```

To build and run immediately:
```powershell
./build.ps1 run
```

### Linux/macOS
```bash
cd cmd/client
go build -o ../../bin/filezap
```

## Usage

1. **Split File**: 
   - Select a file to split
   - Choose an output directory
   - Set chunk size (default: 1MB)
   - Click "Split File"

2. **Join File**:
   - Select a .zap file
   - Choose an output directory
   - Click "Join File"

3. **Network**:
   - Enter the validator server address
   - Click "Connect" to join the network
   - The client will maintain connection and update available files

## Features

- Cross-platform GUI using Fyne toolkit
- File splitting with encryption
- File joining with decryption
- Network integration for distributed file sharing
- Automatic chunk validation
- Progress feedback for operations




`$mingwPaths = @( "C:\mingw64\bin", "C:\msys64\mingw64\bin", "C:\Program Files\mingw-w64\x86_64-8.1.0-posix-seh-rt_v6-rev0\mingw64\bin" ); foreach ($path in $mingwPaths) { if (Test-Path $path) { $env:Path += ";$path"; Write-Host "Added $path to PATH"; Write-Host "Testing GCC..."; gcc --version; break; } }`

`Set-Location -Path Client; $env:CGO_ENABLED="1"; $env:FYNE_RENDER="software"; go build -tags "no_native_menus" -ldflags="-H windowsgui" -v -o bin/filezap.exe cmd/client/main.go cmd/client/main_windows.go`
