#!/bin/bash
# Sync pre-commit additional_dependencies with frontend package.json versions
# This ensures pre-commit uses the same Biome version as your project

set -e

FRONTEND_DIR="frontend"
PRECOMMIT_FILE=".pre-commit-config.yaml"

echo "Checking frontend package versions..."

# Extract Biome version from package.json
cd "$FRONTEND_DIR"
BIOME_VERSION=$(npm list @biomejs/biome --depth=0 2>/dev/null | grep @biomejs/biome@ | sed 's/.*@biomejs\/biome@//' | sed 's/ .*//' || echo "")
cd ..

echo ""
echo "Frontend package versions:"
echo "  @biomejs/biome: $BIOME_VERSION"
echo ""

echo "Current .pre-commit-config.yaml versions:"
grep -A 3 "biomejs/pre-commit" "$PRECOMMIT_FILE" | sed 's/^[ \t]*/  /'
echo ""

echo "To update .pre-commit-config.yaml, run:"
echo ""
echo "  Edit .pre-commit-config.yaml and set:"
echo "    rev: v$BIOME_VERSION"
echo "    additional_dependencies: [\"@biomejs/biome@$BIOME_VERSION\"]"
echo ""
echo "Then run: pre-commit clean && pre-commit install --install-hooks"
