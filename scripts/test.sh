#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

for module in . examples; do
  (
    cd "$module"
    go vet ./...
    go test -race -count=1 ./...
  )
done
