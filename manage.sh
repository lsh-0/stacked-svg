#!/bin/bash

set -e

case "$1" in
    build)
        go build -o svg-stacker
        ;;
    test)
        echo "Running tests with coverage..."
        go test -v -coverprofile=coverage.out -covermode=atomic ./...
        echo ""
        echo "Coverage summary:"
        go tool cover -func=coverage.out | tail -1
        echo ""
        echo "To view detailed coverage report, run: go tool cover -html=coverage.out"
        ;;
    generate)
        if [ -z "$2" ]; then
            echo "Usage: $0 generate <directory> [options...]" >&2
            exit 1
        fi
        dir="$2"
        if [ ! -d "$dir" ]; then
            echo "Error: Directory '$dir' does not exist" >&2
            exit 1
        fi
        if [ ! -f "./svg-stacker" ]; then
            go build -o svg-stacker
        fi
        # Pass directory and all remaining arguments to svg-stacker
        shift 2
        ./svg-stacker "$dir" "$@"
        ;;
    release)
        echo "Building release binaries..."
        mkdir -p release

        LDFLAGS="-s -w"
        BUILDFLAGS="-trimpath"

        # Linux amd64
        GOOS=linux GOARCH=amd64 go build $BUILDFLAGS -ldflags="$LDFLAGS" -o release/svg-stacker-linux-amd64
        echo "Built release/svg-stacker-linux-amd64"

        # Linux arm64
        GOOS=linux GOARCH=arm64 go build $BUILDFLAGS -ldflags="$LDFLAGS" -o release/svg-stacker-linux-arm64
        echo "Built release/svg-stacker-linux-arm64"

        # macOS amd64 (Intel)
        GOOS=darwin GOARCH=amd64 go build $BUILDFLAGS -ldflags="$LDFLAGS" -o release/svg-stacker-darwin-amd64
        echo "Built release/svg-stacker-darwin-amd64"

        # macOS arm64 (Apple Silicon)
        GOOS=darwin GOARCH=arm64 go build $BUILDFLAGS -ldflags="$LDFLAGS" -o release/svg-stacker-darwin-arm64
        echo "Built release/svg-stacker-darwin-arm64"

        # Windows amd64
        GOOS=windows GOARCH=amd64 go build $BUILDFLAGS -ldflags="$LDFLAGS" -o release/svg-stacker-windows-amd64.exe
        echo "Built release/svg-stacker-windows-amd64.exe"

        # Windows arm64
        GOOS=windows GOARCH=arm64 go build $BUILDFLAGS -ldflags="$LDFLAGS" -o release/svg-stacker-windows-arm64.exe
        echo "Built release/svg-stacker-windows-arm64.exe"

        echo ""
        echo "Release binaries created in release/"
        ls -lh release/
        ;;
    clean)
        echo "Cleaning temporary files and binaries..."
        rm -f svg-stacker
        rm -f *.svg
        rm -rf release/
        echo "Cleaned: binaries, test SVGs, and release directory"
        ;;
    *)
        echo "Usage: $0 {build|test|generate <svg-directory>|release|clean}"
        exit 1
        ;;
esac
