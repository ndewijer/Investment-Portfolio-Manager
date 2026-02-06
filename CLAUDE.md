# Claude Code Instructions

## Development Tools

**This project uses UV for Python package management and task running.**

### Running Python Commands
- Use `uv run python` instead of `python` or `python3`
- Use `uv run pytest` instead of `pytest`
- Examples:
  ```bash
  uv run python script.py
  uv run pytest tests/
  uv run flask materialize-history
  ```

### Backend Development
- All backend Python commands should use `uv run`
- Virtual environments are managed by UV automatically
- No need to activate venv manually

## Memory System

This project uses Claude's `.claudememory/` system for persistent context across sessions.

**Key Documentation Files:**
- `.claudememory/project_context.md` - Technology stack, architecture patterns, database info
- `.claudememory/development_workflow.md` - Development processes and documentation maintenance
- `.claudememory/release_process.md` - PR and release management
- `.claudememory/testing_documentation.md` - Testing strategy and documentation

**Note**: The `.claudememory/` directory is gitignored and managed by Claude Code automatically.
