# Contributing Guidelines

## Development Setup
1. Install pre-commit hooks:
```bash
pip install pre-commit
pre-commit install
```

2. Install development dependencies:
```bash
# Backend
pip install -r dev-requirements.txt

# Frontend
npm install
```

## Code Style
- Frontend: ESLint & Prettier
- Backend: Black & Flake8
- Pre-commit hooks enforce style

## Testing
```bash
# Frontend
npm test

# Backend
pytest
```

## Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Run linting and tests
4. Create a Pull Request

## Code Review
- All PRs require review
- CI must pass
- Documentation must be updated
