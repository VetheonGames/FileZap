# FileZap

FileZap is a secure file splitting and joining utility that enables users to split large files into encrypted chunks and later reassemble them. It provides both a graphical user interface and network validation capabilities.

## Components

### Client
- GUI application built with Fyne
- Provides interface for splitting and joining files
- Supports encrypted chunk management
- Network connectivity for validation

### Divider
- Core file splitting functionality
- Handles file chunking and encryption
- Generates unique IDs for file tracking

### Network Core
- Manages network communication
- Handles client-server interactions
- Provides validation protocol implementation

### Reconstructor
- Handles file reconstruction from chunks
- Verifies chunk integrity
- Manages decryption process

### Validator Server
- Validates file operations
- Tracks chunk availability
- Ensures data integrity

## Features

- Split large files into manageable chunks
- Strong encryption for secure storage
- Graphical user interface for easy operation
- Network validation support
- Cross-platform compatibility
- Configurable chunk sizes
- Automatic chunk verification
- Secure file reconstruction

## Building

### Prerequisites

- Go 1.21 or later
- Fyne dependencies
- For Windows:
  - MinGW-w64 with GCC
  - CGO enabled

### Build Scripts

The project includes several build scripts:
- `build.ps1` - Main PowerShell build script (Windows)
- `build.sh` - Main Bash build script (Linux/MacOS)
- Individual OS-specific scripts in the `scripts` directory

To build the project:

1. Run the appropriate main build script for your OS
2. Select target OS and architecture when prompted
3. Built binaries will be placed in respective `bin` directories

For Windows users:
```powershell
.\build.ps1
```

For Linux/MacOS users:
```bash
./build.sh
```

## Usage

1. Launch the FileZap client application
2. Select the "Split File" tab to divide a file:
   - Choose input file
   - Select output directory
   - Set desired chunk size
   - Click "Split File"
3. Select the "Join File" tab to reconstruct a file:
   - Choose .zap file
   - Select output directory
   - Click "Join File"
4. Use the "Network" tab to connect to a validator server if needed

## License

[License information here]
