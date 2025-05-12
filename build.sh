#!/bin/bash
# Main build script for FileZap

# Function to show menu and get selection
show_menu() {
    title="$1"
    shift
    options=("$@")
    
    echo -e "\n$title\n"
    for i in "${!options[@]}"; do
        echo "$((i+1)). ${options[i]}"
    done
    echo -e "\nQ. Quit"
    
    while true; do
        echo -en "\nSelect an option: "
        read selection
        
        if [[ "$selection" == "Q" ]] || [[ "$selection" == "q" ]]; then
            exit 0
        fi
        
        if [[ "$selection" =~ ^[0-9]+$ ]]; then
            index=$((selection-1))
            if [ "$index" -ge 0 ] && [ "$index" -lt "${#options[@]}" ]; then
                echo "$index"
                return
            fi
        fi
        
        echo "Invalid selection. Please try again."
    done
}

# Make scripts executable
chmod +x ./scripts/build_linux.sh
chmod +x ./scripts/build_mac.sh

# Determine current OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    current_os="macos"
elif [[ "$OSTYPE" == "linux"* ]]; then
    current_os="linux"
else
    echo "Error: Unsupported operating system"
    exit 1
fi

# Set up OS and architecture options
os_options=("Windows" "Linux" "macOS")
arch_options=("amd64 (x64)" "386 (x86)" "arm64")

# Show build system header
echo "FileZap Build System"
echo "==================="

# Get OS selection
os_index=$(show_menu "Select Target Operating System" "${os_options[@]}")
target_os=$(echo "${os_options[$os_index]}" | tr '[:upper:]' '[:lower:]')

# Get architecture selection
arch_index=$(show_menu "Select Target Architecture" "${arch_options[@]}")
case $arch_index in
    0) target_arch="amd64" ;;
    1) target_arch="386" ;;
    2) target_arch="arm64" ;;
esac

echo -e "\nBuilding for $target_os/$target_arch..."

# Execute appropriate build script
if [ "$current_os" == "macos" ]; then
    if [ "$target_os" == "windows" ]; then
        echo "Error: Building for Windows is not supported on macOS"
        exit 1
    fi
    
    if [ "$target_os" == "linux" ]; then
        ./scripts/build_linux.sh "$target_arch"
    else
        ./scripts/build_mac.sh "$target_arch"
    fi
elif [ "$current_os" == "linux" ]; then
    if [ "$target_os" == "windows" ] || [ "$target_os" == "macos" ]; then
        echo "Error: Building for $target_os is not supported on Linux"
        exit 1
    fi
    
    ./scripts/build_linux.sh "$target_arch"
fi

echo -e "\nBuild process completed!"
