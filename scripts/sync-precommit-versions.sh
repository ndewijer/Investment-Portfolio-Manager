#!/bin/bash
# Sync pre-commit additional_dependencies with frontend package.json versions
# This ensures pre-commit uses the same ESLint/Prettier versions as your project

set -e

FRONTEND_DIR="frontend"
PRECOMMIT_FILE=".pre-commit-config.yaml"

echo "🔍 Checking frontend package versions..."

# Extract versions from package.json
cd "$FRONTEND_DIR"

ESLINT_VERSION=$(npm list eslint --depth=0 2>/dev/null | grep eslint@ | sed 's/.*eslint@//' | sed 's/ .*//' || echo "")
ESLINT_REACT_VERSION=$(npm list eslint-plugin-react --depth=0 2>/dev/null | grep eslint-plugin-react@ | sed 's/.*eslint-plugin-react@//' | sed 's/ .*//' || echo "")
ESLINT_HOOKS_VERSION=$(npm list eslint-plugin-react-hooks --depth=0 2>/dev/null | grep eslint-plugin-react-hooks@ | sed 's/.*eslint-plugin-react-hooks@//' | sed 's/ .*//' || echo "")
ESLINT_PRETTIER_VERSION=$(npm list eslint-config-prettier --depth=0 2>/dev/null | grep eslint-config-prettier@ | sed 's/.*eslint-config-prettier@//' | sed 's/ .*//' || echo "")
ESLINT_COMPAT_VERSION=$(npm list @eslint/compat --depth=0 2>/dev/null | grep @eslint/compat@ | sed 's/.*@eslint\/compat@//' | sed 's/ .*//' || echo "")
ESLINT_ESLINTRC_VERSION=$(npm list @eslint/eslintrc --depth=0 2>/dev/null | grep @eslint/eslintrc@ | sed 's/.*@eslint\/eslintrc@//' | sed 's/ .*//' || echo "")
ESLINT_JS_VERSION=$(npm list @eslint/js --depth=0 2>/dev/null | grep @eslint/js@ | sed 's/.*@eslint\/js@//' | sed 's/ .*//' || echo "")
GLOBALS_VERSION=$(npm list globals --depth=0 2>/dev/null | grep globals@ | sed 's/.*globals@//' | sed 's/ .*//' || echo "")
PRETTIER_VERSION=$(npm list prettier --depth=0 2>/dev/null | grep prettier@ | sed 's/.*prettier@//' | sed 's/ .*//' || echo "")

cd ..

echo ""
echo "📦 Frontend package versions:"
echo "  eslint: $ESLINT_VERSION"
echo "  eslint-plugin-react: $ESLINT_REACT_VERSION"
echo "  eslint-plugin-react-hooks: $ESLINT_HOOKS_VERSION"
echo "  eslint-config-prettier: $ESLINT_PRETTIER_VERSION"
echo "  @eslint/compat: $ESLINT_COMPAT_VERSION"
echo "  @eslint/eslintrc: $ESLINT_ESLINTRC_VERSION"
echo "  @eslint/js: $ESLINT_JS_VERSION"
echo "  globals: $GLOBALS_VERSION"
echo "  prettier: $PRETTIER_VERSION"
echo ""

echo "📝 Current .pre-commit-config.yaml versions:"
grep -A 10 "additional_dependencies:" "$PRECOMMIT_FILE" | grep -E "eslint|globals" | sed 's/^[ \t]*/  /'
echo ""

echo "💡 To update .pre-commit-config.yaml, run:"
echo ""
echo "  Edit .pre-commit-config.yaml and set:"
echo "    - eslint@$ESLINT_VERSION"
echo "    - eslint-plugin-react@$ESLINT_REACT_VERSION"
echo "    - eslint-plugin-react-hooks@$ESLINT_HOOKS_VERSION"
echo "    - eslint-config-prettier@$ESLINT_PRETTIER_VERSION"
echo "    - \"@eslint/compat@$ESLINT_COMPAT_VERSION\""
echo "    - \"@eslint/eslintrc@$ESLINT_ESLINTRC_VERSION\""
echo "    - \"@eslint/js@$ESLINT_JS_VERSION\""
echo "    - globals@$GLOBALS_VERSION"
echo ""
echo "  And in the prettier section:"
echo "    rev: v$PRETTIER_VERSION"
echo ""
echo "Then run: pre-commit clean && pre-commit install --install-hooks"
