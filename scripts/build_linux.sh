#!/bin/bash
# Linux build script for FileZap

# Default architecture
ARCH=${1:-amd64}

# Set required environment variables
export CGO_ENABLED=1
export FYNE_RENDER=software
export GOOS=linux
export GOARCH=$ARCH

# Function to build a component
build_component() {
    local name=$1
    local main_file=$2
    local flags=$3

    echo -e "\nBuilding $name..."
    cd "$name" || exit 1

    # Create bin directory if it doesn't exist
    mkdir -p bin

    # Construct build command
    output="bin/${name,,}"  # Convert to lowercase
    output=${output// /}    # Remove spaces
    build_cmd="go build -v -o $output"

    if [ -n "$flags" ]; then
        build_cmd="$build_cmd $flags"
    fi

    build_cmd="$build_cmd $main_file"

    # Execute build
    echo "Executing: $build_cmd"
    eval "$build_cmd"

    if [ $? -ne 0 ]; then
        echo "Build failed for $name"
        exit 1
    fi

    cd ..
}

# Build each component
# Client requires special flags for GUI
build_component "Client" "cmd/client/main.go" "-tags no_native_menus"
build_component "Divider" "cmd/divider/main.go" ""
build_component "Network Core" "cmd/networkcore/main.go" ""
build_component "Reconstructor" "cmd/reconstructor/main.go" ""
build_component "Validator Server" "cmd/validatorserver/main.go" ""

echo -e "\nAll components built successfully!"
