# uv Migration Guide

## Quick Start for Existing Developers

### 1. Install uv
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

### 2. Sync dependencies
```bash
# From project root
uv sync --frozen
```

### 3. Update your workflow
| Old (pip) | New (uv) |
|-----------|----------|
| `source backend/.venv/bin/activate` | Not needed! |
| `pip install -r dev-requirements.txt` | `uv sync --frozen` |
| `pytest tests/` | `uv run pytest backend/tests/` |
| `flask run` | `cd backend && uv run flask run` |
| `ruff check .` | `uv run ruff check backend/` |

## What Changed?

### Dependency Files
- **Primary source**: `pyproject.toml` at project root
  - `[project.dependencies]`: Production (pinned ==)
  - `[dependency-groups.dev]`: Development (flexible ranges)
- **Lockfile**: `uv.lock` (like package-lock.json for npm)
- **Legacy files**: `backend/requirements.txt` and `backend/dev-requirements.txt` kept temporarily, deprecated

### Benefits
- ⚡ 10-100x faster installation
- 🔒 Reproducible builds with uv.lock
- 🎯 Single source of truth (pyproject.toml)
- 🚫 No manual venv activation

## FAQ

**Q: Can I still use pip?**
A: Temporarily yes (requirements.txt maintained), but deprecated. Will be removed in v1.4.0.

**Q: Do I need to activate .venv?**
A: No! Use `uv run` instead. It automatically uses the correct environment.

**Q: How do I add a dependency?**
A:
1. Edit `pyproject.toml` (production = pinned, dev = range)
2. Run `uv lock`
3. Run `uv sync --frozen`

**Q: What if uv doesn't work?**
A: Fallback to pip with `backend/dev-requirements.txt` and report the issue.

## Timeline
- **v1.3.2**: Migration complete, pip deprecated
- **v1.4.0**: Remove requirements.txt files

## Resources
- [uv docs](https://docs.astral.sh/uv/)
- [uv GitHub](https://github.com/astral-sh/uv)
