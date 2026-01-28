#!/usr/bin/env bash
set -euo pipefail

# Release script for creating prerelease/test version tags
# Usage: hack/release-prerelease.sh <version>
# Examples:
#   hack/release-prerelease.sh v0.0.0-test.1    # Test release
#   hack/release-prerelease.sh v0.1.0-rc.1      # Prerelease

VERSION="${1:-}"

if [ -z "$VERSION" ]; then
  echo "Error: Version required"
  echo "Usage: $0 <version>"
  echo ""
  echo "Examples:"
  echo "  $0 v0.0.0-test.1    # Test release"
  echo "  $0 v0.1.0-rc.1      # Prerelease"
  exit 1
fi

# Validate version format (must have a prerelease suffix)
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*$ ]]; then
  echo "Error: Invalid prerelease version format: $VERSION"
  echo "Must match: v<major>.<minor>.<patch>-<prerelease>"
  echo "For stable releases, use hack/release.sh instead"
  exit 1
fi

# Check we're on main branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$CURRENT_BRANCH" != "main" ]; then
  echo "Error: Must be on main branch (currently on: $CURRENT_BRANCH)"
  echo "Run: git checkout main"
  exit 1
fi

# Check working directory is clean
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: Working directory has uncommitted changes"
  git status --short
  exit 1
fi

# Check we're up to date with origin
echo "Fetching from origin..."
git fetch origin main

LOCAL=$(git rev-parse main)
REMOTE=$(git rev-parse origin/main)

if [ "$LOCAL" != "$REMOTE" ]; then
  echo "Error: Local main is not up to date with origin/main"
  echo "Run: git pull origin main"
  exit 1
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "Error: Tag $VERSION already exists"
  exit 1
fi

echo ""
echo "======================================"
echo "Creating prerelease tag: $VERSION"
echo "======================================"
echo "Branch: $CURRENT_BRANCH"
echo "Commit: $(git rev-parse --short HEAD)"
echo ""

# Ask for confirmation
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Aborted"
  exit 1
fi

git tag -a "$VERSION" -m "Prerelease $VERSION"
git push origin "$VERSION"

echo ""
echo "======================================"
echo "✓ Prerelease tag pushed: $VERSION"
echo "======================================"
echo ""
echo "Monitor progress:"
echo "  https://github.com/mirendev/runtime/actions"
echo ""
