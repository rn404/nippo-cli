#!/usr/bin/env bash
# Create a release PR that bumps cmd/sava/version.txt.
# Merging the PR is the release approval (see release.yml).
# Usage: create-release-pr.sh <patch|minor|major>
# Requires: gh (authenticated), git push permission.
set -euo pipefail

cd "$(dirname "$0")/.."
bump="${1:?usage: create-release-pr.sh <patch|minor|major>}"

next="$(bash scripts/bump-version.sh "$bump")"
base_sha="$(git rev-parse HEAD)"
repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"

git switch -c "release/$next"
echo "$next" > cmd/sava/version.txt
git commit -am "release: $next"
git push -u origin "release/$next"

# Prefill the PR body with auto-generated notes; edit it before
# merging — the merged body becomes the release notes.
notes="$(gh api "repos/$repo/releases/generate-notes" \
  -f tag_name="$next" -f target_commitish="$base_sha" --jq .body)"
gh pr create --title "Release $next" --body "$notes"
