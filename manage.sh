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
            echo "Usage: $0 generate <directory> [output.svg]" >&2
            exit 1
        fi
        if [ ! -d "$2" ]; then
            echo "Error: Directory '$2' does not exist" >&2
            exit 1
        fi
        if [ ! -f "./svg-stacker" ]; then
            go build -o svg-stacker
        fi
        if [ -n "$3" ]; then
            ./svg-stacker "$2" -o "$3"
        else
            ./svg-stacker "$2"
        fi
        ;;
    *)
        echo "Usage: $0 {build|test|generate <svg-directory>}"
        exit 1
        ;;
esac
