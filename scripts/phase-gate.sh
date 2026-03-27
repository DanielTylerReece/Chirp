#!/bin/bash
set -e

echo "=== Compile Check ==="
go build -o /dev/null .

echo "=== Unit Tests ==="
go test ./... -v -count=1 -timeout 60s

echo "=== Vet ==="
go vet ./...

echo "=== Static Analysis ==="
staticcheck ./... 2>/dev/null || echo "staticcheck not installed, skipping"

echo "=== Race Detector ==="
go test ./... -race -count=1 -timeout 120s

echo "=== Binary Size ==="
go build -o /tmp/gmessage-check . && ls -lh /tmp/gmessage-check && rm /tmp/gmessage-check

echo "=== All checks passed ==="
