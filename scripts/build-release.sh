#!/usr/bin/env bash
# Cross-compile release binaries into dist/ with sha256 checksums.
# Usage: build-release.sh <tag>
set -euo pipefail

cd "$(dirname "$0")/.."
tag="${1:?usage: build-release.sh <tag>}"

rm -rf dist
mkdir -p dist

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
  export GOOS="${target%/*}" GOARCH="${target#*/}"
  name="sava_${tag}_${GOOS}_${GOARCH}"
  CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" \
    -o "dist/${name}/sava" ./cmd/sava
  tar -czf "dist/${name}.tar.gz" -C dist "$name"
  rm -r "dist/${name:?}"
done

(cd dist && shasum -a 256 ./* > checksums.txt)
