#!/bin/bash

# YouTube Summarizer - Cross-Platform Build Script

set -e

VERSION=${1:-"1.0.0"}
BUILD_DIR="builds"
APP_NAME="youtube-summarizer"

echo "🚀 Building YouTube Summarizer v$VERSION"

# Clean previous builds
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Build for different platforms
echo "📦 Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-windows-amd64.exe ./cmd/summarizer

echo "📦 Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-darwin-amd64 ./cmd/summarizer

echo "📦 Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-darwin-arm64 ./cmd/summarizer

echo "📦 Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-linux-amd64 ./cmd/summarizer

echo "📦 Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$VERSION" -o $BUILD_DIR/${APP_NAME}-linux-arm64 ./cmd/summarizer

# Copy configuration files to build directory
echo "📄 Copying configuration files..."
cp -r configs $BUILD_DIR/
cp README.md $BUILD_DIR/
cp .env.example $BUILD_DIR/ 2>/dev/null || echo "⚠️  .env.example not found, skipping"

# Create distribution packages
echo "📦 Creating distribution packages..."
cd $BUILD_DIR

for binary in ${APP_NAME}-*; do
    if [[ -f "$binary" ]]; then
        platform=$(echo $binary | sed "s/${APP_NAME}-//")
        
        # Create package directory
        package_dir="${APP_NAME}-${platform}-v${VERSION}"
        mkdir -p $package_dir
        
        # Copy binary and config files
        cp $binary $package_dir/
        cp -r configs $package_dir/
        cp README.md $package_dir/
        cp .env.example $package_dir/ 2>/dev/null || true
        
        # Create archive based on platform
        if [[ $platform == *"windows"* ]]; then
            zip -r ${package_dir}.zip $package_dir
            echo "✅ Created ${package_dir}.zip"
        else
            tar -czf ${package_dir}.tar.gz $package_dir
            echo "✅ Created ${package_dir}.tar.gz"
        fi
        
        # Clean up package directory
        rm -rf $package_dir
    fi
done

cd ..

echo ""
echo "🎉 Build completed successfully!"
echo "📂 Distribution packages created in: $BUILD_DIR/"
echo ""
echo "Built binaries:"
ls -la $BUILD_DIR/${APP_NAME}-* | grep -v ".zip\|.tar.gz"
echo ""
echo "Distribution packages:"
ls -la $BUILD_DIR/*.{zip,tar.gz} 2>/dev/null || echo "No packages created"

echo ""
echo "🚀 To run the application:"
echo "   1. Extract the appropriate package for your platform"
echo "   2. Copy .env.example to .env and configure API keys"
echo "   3. Run the binary with: ./${APP_NAME}-<platform>"