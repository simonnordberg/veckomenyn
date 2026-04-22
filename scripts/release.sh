#!/usr/bin/env bash
set -euo pipefail

# Bump version and create a GitHub release.
# Usage: ./scripts/release.sh [major|minor|patch]
#   Defaults to patch.

BUMP="${1:-patch}"

# Get latest tag, default to v0.0.0 if none exists
LATEST=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
IFS='.' read -r MAJOR MINOR PATCH <<< "${LATEST#v}"

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
  *) echo "Usage: $0 [major|minor|patch]"; exit 1 ;;
esac

TAG="v${MAJOR}.${MINOR}.${PATCH}"

echo "${LATEST} → ${TAG}"
git tag "$TAG"
git push origin "$TAG"
echo "Release ${TAG} pushed — GoReleaser will build binaries."
