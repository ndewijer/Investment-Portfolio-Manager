# Test Coverage Monitoring Guide

## Viewing Coverage

### 1. Terminal Output (Default)
Run tests and see coverage in the terminal:
```bash
npm test
```

Look for the **"All files"** row at the top:
```
------------------------------|---------|----------|---------|---------|
File                          | % Stmts | % Branch | % Funcs | % Lines |
------------------------------|---------|----------|---------|---------|
All files                     |   90.74 |    84.83 |    81.3 |   93.11 | ← TOTAL COVERAGE
```

### 2. HTML Report (Interactive)
Generate an interactive HTML report:
```bash
npm run test:coverage
```

Then open `coverage/index.html` in your browser:
```bash
open coverage/index.html
```

Features:
- Click-through to see uncovered lines
- Visual highlighting of covered/uncovered code
- Drill down into specific files/directories
- Easy to share with team

### 3. JSON Summary (Machine-Readable)
Generate a machine-readable summary:
```bash
npm run test:coverage:summary
```

Creates `coverage/coverage-summary.json`:
```json
{
  "total": {
    "lines": { "total": 100, "covered": 93, "skipped": 0, "pct": 93.11 },
    "statements": { "total": 108, "covered": 98, "skipped": 0, "pct": 90.74 },
    "functions": { "total": 23, "covered": 18, "skipped": 0, "pct": 81.3 },
    "branches": { "total": 58, "covered": 49, "skipped": 0, "pct": 84.83 }
  }
}
```

## Alerting on Coverage

### 1. ✅ Mandatory Thresholds (Currently Configured)

Jest will **fail the test run** if coverage drops below:
- **Branches**: 80%
- **Functions**: 75%
- **Lines**: 85%
- **Statements**: 85%

Configuration in `package.json`:
```json
"coverageThreshold": {
  "global": {
    "branches": 80,
    "functions": 75,
    "lines": 85,
    "statements": 85
  }
}
```

**When it fails:**
```
Jest: "global" coverage threshold for statements (85%) not met: 82.5%
Jest: "global" coverage threshold for lines (85%) not met: 83.2%
```

### 2. Pre-commit Hook Integration

Add to `.pre-commit-config.yaml` (if using pre-commit):
```yaml
repos:
  - repo: local
    hooks:
      - id: jest-coverage
        name: Jest Coverage Check
        entry: bash -c 'cd frontend && npm test -- --testPathIgnorePatterns=e2e'
        language: system
        pass_filenames: false
        always_run: true
```

Or create `.husky/pre-commit` (if using Husky):
```bash
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

cd frontend && npm test -- --testPathIgnorePatterns=e2e
```

### 3. CI/CD Pipeline Alerts

#### GitHub Actions Example
```yaml
name: Test Coverage

on: [push, pull_request]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install dependencies
        run: cd frontend && npm ci
      - name: Run tests with coverage
        run: cd frontend && npm test -- --testPathIgnorePatterns=e2e
      - name: Upload coverage reports
        uses: codecov/codecov-action@v3
        with:
          directory: ./frontend/coverage
          fail_ci_if_error: true
```

#### GitLab CI Example
```yaml
test:coverage:
  stage: test
  script:
    - cd frontend
    - npm ci
    - npm test -- --testPathIgnorePatterns=e2e
  coverage: '/All files[^|]*\|[^|]*\s+([\d\.]+)/'
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: frontend/coverage/cobertura-coverage.xml
```

### 4. Coverage Badges

#### Using Codecov
1. Sign up at https://codecov.io
2. Add your repository
3. Get badge markdown:
```markdown
[![codecov](https://codecov.io/gh/username/repo/branch/main/graph/badge.svg)](https://codecov.io/gh/username/repo)
```

#### Using Shields.io (Manual)
```markdown
![Coverage](https://img.shields.io/badge/coverage-93%25-brightgreen)
```

### 5. Coverage Trend Monitoring

#### Option A: Codecov (Recommended)
- Automatic trend graphs
- PR comments showing coverage changes
- Email alerts on coverage drops
- Free for open source

#### Option B: Custom Script
Create `scripts/check-coverage-change.sh`:
```bash
#!/bin/bash

# Get current coverage
CURRENT=$(npm test -- --testPathIgnorePatterns=e2e --coverageReporters=json-summary 2>&1 | grep -o 'All files.*' | awk '{print $4}')

# Compare with baseline (stored in git)
BASELINE=$(cat coverage-baseline.txt)

if (( $(echo "$CURRENT < $BASELINE" | bc -l) )); then
  echo "❌ Coverage decreased from $BASELINE% to $CURRENT%"
  exit 1
else
  echo "✅ Coverage maintained: $CURRENT%"
  exit 0
fi
```

### 6. Slack/Discord Notifications

#### Using GitHub Actions + Slack
```yaml
- name: Notify Slack on coverage failure
  if: failure()
  uses: slackapi/slack-github-action@v1
  with:
    payload: |
      {
        "text": "❌ Test coverage dropped below threshold in ${{ github.repository }}",
        "blocks": [
          {
            "type": "section",
            "text": {
              "type": "mrkdwn",
              "text": "*Coverage Alert*\nTest coverage is below the required threshold.\n*Repository:* ${{ github.repository }}\n*Branch:* ${{ github.ref }}"
            }
          }
        ]
      }
  env:
    SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK_URL }}
```

## Current Coverage Status

**Last Updated**: 2024-12-18

| Metric | Current | Threshold | Status |
|--------|---------|-----------|--------|
| Lines | 93.11% | 85% | ✅ +8.11% |
| Statements | 90.74% | 85% | ✅ +5.74% |
| Branches | 84.83% | 80% | ✅ +4.83% |
| Functions | 81.3% | 75% | ✅ +6.3% |

## What's Excluded from Coverage

To keep reports focused on testable business logic:
- Pages (`src/pages/**`) - covered by E2E tests
- Infrastructure (`src/config.js`, `src/utils/api.js`)
- UI-heavy components (ValueChart, FilterPopup, etc.)
- Layout components (Navigation, StatusTab, etc.)
- Barrel files (`**/index.js`)

See `package.json` → `jest.collectCoverageFrom` for full list.

## Troubleshooting

### "Coverage threshold not met" error
1. Run `npm test` to see current coverage
2. Identify files below threshold in the report
3. Add tests for uncovered code OR
4. Exclude files from coverage if they're infrastructure/UI

### Coverage reports not generating
1. Ensure `coverage` directory exists and is writable
2. Check `.gitignore` includes `/coverage/` (reports shouldn't be committed)
3. Run `npm test -- --coverage --verbose` for detailed output

### Pre-commit hook failing
1. Test manually: `npm test -- --testPathIgnorePatterns=e2e`
2. If passing manually but failing in hook, check Node version
3. Ensure all dependencies are installed: `npm ci`

## Best Practices

1. **Run tests before committing**: `npm test`
2. **Check coverage for new code**: Aim for 90%+ on new files
3. **Don't game the numbers**: Write meaningful tests, not just for coverage
4. **Review coverage reports**: Use HTML report to find gaps
5. **Update thresholds carefully**: Only increase, never decrease
6. **Document exclusions**: Add comments explaining why files are excluded

---

**Version**: 1.3.5+
**Last Updated**: 2025-12-18
**Maintainer**: @ndewijer
