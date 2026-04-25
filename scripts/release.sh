#!/usr/bin/env bash
# Bump version, sync compose+docs, commit, tag.
#
# Usage:
#   bash scripts/release.sh           # patch bump (default)
#   bash scripts/release.sh patch
#   bash scripts/release.sh minor
#   bash scripts/release.sh major
#
# The next version is derived from the latest annotated tag. Patch bumps
# are docs no-ops because the compose channel pin doesn't change. Minor
# and major bumps rewrite docker-compose.yml's image pin and the README
# upgrade-channel examples to point at the new minor.
#
# Does NOT push. The push command is printed at the end.

set -euo pipefail

BUMP="${1:-patch}"
case "$BUMP" in
  patch|minor|major) ;;
  *)
    echo "usage: $0 [patch|minor|major]" >&2
    exit 1
    ;;
esac

if ! git diff --quiet HEAD || [ -n "$(git status --porcelain)" ]; then
  echo "working tree must be clean. Commit or stash changes first." >&2
  exit 1
fi

LATEST=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
LATEST=${LATEST#v}
if ! [[ "$LATEST" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "no semver tag found in git history (got '${LATEST:-<none>}')" >&2
  echo "create v0.0.0 manually for the first release, or pass a base tag" >&2
  exit 1
fi

IFS=. read -r MAJOR MINOR PATCH <<<"$LATEST"
case "$BUMP" in
  patch) PATCH=$((PATCH + 1)) ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
esac
VERSION="${MAJOR}.${MINOR}.${PATCH}"

if git rev-parse --verify "v${VERSION}" >/dev/null 2>&1; then
  echo "tag v${VERSION} already exists" >&2
  exit 1
fi

NEW_MINOR="${VERSION%.*}"
PREV_MINOR=$(perl -ne 'if (/ghcr\.io\/[^:\s]+:(\d+\.\d+)\b/) { print $1; exit }' docker-compose.yml)

if [ -z "$PREV_MINOR" ]; then
  echo "could not find an existing X.Y channel pin in docker-compose.yml" >&2
  exit 1
fi

echo "latest tag      v${LATEST}"
echo "bump            ${BUMP}"
echo "next version    v${VERSION} (channel :${NEW_MINOR})"
echo

if [ "$NEW_MINOR" != "$PREV_MINOR" ]; then
  echo "bumping channel references ${PREV_MINOR} -> ${NEW_MINOR}"

  # Files that may carry channel references. Globbing docs/*.md catches
  # the deploy/upgrade/backups guides; add new locations here as needed.
  files=(docker-compose.yml README.md docs/*.md)

  perl -i -pe "s|veckomenyn:\Q${PREV_MINOR}\E\b|veckomenyn:${NEW_MINOR}|g" "${files[@]}"
  perl -i -pe "s|the \Q${PREV_MINOR}\E line|the ${NEW_MINOR} line|g" "${files[@]}"
  perl -i -pe "s|:\Q${PREV_MINOR}\E\.\d+\b|:${NEW_MINOR}.0|g" "${files[@]}"
  perl -i -pe "s|\`:\Q${PREV_MINOR}\E\`|\`:${NEW_MINOR}\`|g" "${files[@]}"
  perl -i -pe "s|\Q${PREV_MINOR}\E\.x|${NEW_MINOR}.x|g" "${files[@]}"

  if ! git diff --quiet; then
    git add "${files[@]}"
    git commit -m "chore: bump compose+docs to :${NEW_MINOR} for v${VERSION}"
    echo
    echo "committed channel bump:"
    git --no-pager log -1 --stat
  else
    echo "no doc changes needed (already in sync)"
  fi
else
  echo "patch bump within :${NEW_MINOR}, no doc changes needed"
fi

echo
git tag -a "v${VERSION}" -m "v${VERSION}"
echo "tagged v${VERSION}"

cat <<HINT

Push when ready:
  git push origin main && git push origin v${VERSION}

The release workflow validates the compose channel matches the tag,
then publishes ghcr.io/simonnordberg/veckomenyn:{${VERSION},${NEW_MINOR},${NEW_MINOR%.*},latest}.
HINT
