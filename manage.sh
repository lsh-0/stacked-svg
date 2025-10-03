#!/bin/bash

set -e

case "$1" in
    build)
        go build -o svg-stacker
        ;;
    test)
        go test ./...
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
    *)
        echo "Usage: $0 {build|test|generate <svg-directory>}"
        exit 1
        ;;
esac
