#!/bin/bash

# Terminal Emulator Build Script
# Builds all variants or specific ones based on configuration

# Default variants to build (can be overridden)
DEFAULT_VARIANTS="sonoma-dusk-alpha sonoma-sky-alpha glm-4.5 grok-code-fast-1"

# Use default variants (configuration now in config.toml)
VARIANTS=$DEFAULT_VARIANTS

echo " Building Terminal Emulator Variants and Test Suite"
echo "Variants to build: $VARIANTS"
echo ""

# Create directories
mkdir -p test/bin

# Build each variant
for variant in $VARIANTS; do
    if [ -d "$variant" ]; then
        echo "Building $variant..."
        cd "$variant"
        
        # Check if go.mod exists
        if [ -f "go.mod" ]; then
            go build -o "../test/bin/${variant}.exe"
            if [ $? -eq 0 ]; then
                echo "[OK] Successfully built $variant"
            else
                echo "[ERROR] Failed to build $variant"
            fi
        else
            echo "[WARN]  No go.mod found in $variant directory"
        fi
        
        cd ..
    else
        echo "[WARN]  Directory $variant not found"
    fi
done

echo ""
echo "Building test suite..."
cd test
go build -o bin/test_suite.exe *.go
if [ $? -eq 0 ]; then
    echo "[OK] Successfully built test_suite"
else
    echo "[ERROR] Failed to build test_suite"
fi
cd ..

echo ""
echo " Build complete!"
echo ""
echo "Built executables:"
ls -la test/bin/*.exe 2>/dev/null || echo "No executables found"

echo ""
echo "To configure test settings, edit test/config.toml"