#!/bin/bash

# Script to bump version across all project files
# Usage: ./scripts/bump-version.sh <new-version>
# Example: ./scripts/bump-version.sh 0.2.0

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <new-version>"
    echo "Example: $0 0.2.0"
    exit 1
fi

NEW_VERSION=$1

# Validate version format (basic semver check)
if ! [[ $NEW_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "Error: Version must follow semantic versioning format (e.g., 1.2.3 or 1.2.3-beta.1)"
    exit 1
fi

echo "Bumping version to $NEW_VERSION..."

# Update VERSION file
echo "$NEW_VERSION" > VERSION
echo "✓ Updated VERSION file"

# Update frontend package.json
if [ -f "frontend/package.json" ]; then
    sed -i "s/\"version\": \".*\"/\"version\": \"$NEW_VERSION\"/" frontend/package.json
    echo "✓ Updated frontend/package.json"
fi

# Update CHANGELOG.md (add unreleased marker if not present)
if [ -f "CHANGELOG.md" ]; then
    # Check if there's an [Unreleased] section
    if grep -q "\[Unreleased\]" CHANGELOG.md; then
        # Get today's date
        TODAY=$(date +%Y-%m-%d)

        # Replace [Unreleased] with the new version
        sed -i "s/## \[Unreleased\]/## [$NEW_VERSION] - $TODAY/" CHANGELOG.md

        # Add new [Unreleased] section at the top
        sed -i "/## \[$NEW_VERSION\]/i ## [Unreleased]\n" CHANGELOG.md

        echo "✓ Updated CHANGELOG.md"
    else
        echo "⚠ No [Unreleased] section found in CHANGELOG.md - skipping"
    fi
fi

echo ""
echo "Version bumped to $NEW_VERSION successfully!"
echo ""
echo "Next steps:"
echo "  1. Review the changes: git diff"
echo "  2. Commit the changes: git add -A && git commit -m \"Bump version to $NEW_VERSION\""
echo "  3. Create a git tag: git tag -a v$NEW_VERSION -m \"Release v$NEW_VERSION\""
echo "  4. Push changes and tag: git push && git push --tags"
