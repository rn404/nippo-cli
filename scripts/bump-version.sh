#!/usr/bin/env bash
# Print the next version for the given bump level.
# Usage: bump-version.sh <patch|minor|major>
set -euo pipefail

VERSION_FILE="$(dirname "$0")/../cmd/sava/version.txt"
bump="${1:?usage: bump-version.sh <patch|minor|major>}"

current="$(cat "$VERSION_FILE")"
IFS=. read -r major minor patch <<< "${current#v}"

case "$bump" in
  major) echo "v$((major + 1)).0.0" ;;
  minor) echo "v${major}.$((minor + 1)).0" ;;
  patch) echo "v${major}.${minor}.$((patch + 1))" ;;
  *) echo "unknown bump level: $bump" >&2; exit 1 ;;
esac
