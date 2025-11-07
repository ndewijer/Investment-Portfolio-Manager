# API Documentation Generation Plan

## Overview

This document outlines a plan to create an automated API documentation generator for the Investment Portfolio Manager. The goal is to generate comprehensive, accurate API documentation directly from the codebase, reducing manual maintenance and ensuring documentation stays in sync with code.

## Current State Analysis

### Existing Route Structure

The application has the following route modules:
- `developer_routes.py` - Development/debugging endpoints
- `dividend_routes.py` - Dividend management
- `fund_routes.py` - Fund management
- `ibkr_routes.py` - IBKR integration
- `portfolio_routes.py` - Portfolio management
- `system_routes.py` - System health/version
- `transaction_routes.py` - Transaction management

### Documentation Pattern in Code

Routes are defined using Flask decorators and have varying levels of documentation:

```python
@ibkr.route("/ibkr/inbox/<transaction_id>/allocate", methods=["POST"])
def allocate_transaction(transaction_id):
    """
    Process IBKR transaction with allocations.

    Request body:
        {
            "allocations": [
                {
                    "portfolio_id": "string",
                    "percentage": number
                },
                ...
            ]
        }

    Args:
        transaction_id: Transaction ID

    Returns:
        JSON response with processing results
    """
```

Some routes use Flask MethodView classes:
```python
class IBKRConfigAPI(MethodView):
    """RESTful API for IBKR configuration management."""

    def get(self):
        """Get IBKR configuration status."""

    def post(self):
        """Create or update IBKR configuration."""
```

### Current Documentation Gaps

- **Inconsistent Coverage**: Only 3-4 IBKR endpoints documented out of 19 total
- **Scattered Documentation**: Some in user guides, some in release notes, none comprehensive
- **No Unified Format**: Different documentation styles across files
- **Manual Updates Required**: Documentation doesn't auto-sync with code changes
- **Missing Details**: Many endpoints lack request/response examples, error codes, validation rules

### Code Implementation Inconsistencies

**Current State**: Routes are implemented using multiple patterns, which creates maintenance challenges:

1. **Function-based routes with decorators**:
   ```python
   @ibkr.route("/ibkr/inbox/<id>/allocate", methods=["POST"])
   def allocate_transaction(id):
       """Docstring"""
       # Implementation
   ```

2. **Class-based routes with MethodView**:
   ```python
   class IBKRConfigAPI(MethodView):
       def get(self):
           """Docstring"""
       def post(self):
           """Docstring"""
   ```

3. **Mixed registration patterns**:
   - Some use `@blueprint.route()` decorator
   - Some use `blueprint.add_url_rule()` with MethodView
   - Inconsistent use of helper decorators (`@track_request`, etc.)

**Impact on Documentation Generation**:
- Parser must handle multiple patterns
- Increases complexity of extraction logic
- Higher chance of missing edge cases
- More difficult to enforce consistent documentation standards

**Recommendation for Future**:

When implementing the documentation generator, we should:

1. **Document the current patterns** (Phase 1-2)
   - Build parser that handles all existing patterns
   - Generate initial documentation to establish baseline

2. **Evaluate standardization** (Phase 3)
   - Assess which pattern best serves the project
   - Consider factors: consistency, testability, documentation ease
   - Decide on single recommended pattern

3. **Create migration plan** (Phase 4)
   - Document chosen pattern in `CONTRIBUTING.md`
   - Create examples and templates
   - Optionally create refactoring script to convert existing routes
   - Apply to new endpoints going forward

4. **Gradual migration** (Phase 5+)
   - Convert routes opportunistically during feature work
   - No need for massive refactor unless necessary
   - Aim for consistency over time

**Pattern Recommendation** (preliminary):

After implementation experience, we should choose **ONE** of these approaches:

**Option A: Pure Function-Based** (current majority pattern)
- Pros: Simple, explicit, easy to understand
- Cons: More repetitive for REST resources, harder to share logic

**Option B: Class-Based MethodView** (RESTful standard)
- Pros: Natural for REST resources, shares validation/auth logic
- Cons: More abstraction, slightly more complex

**Option C: Hybrid** (current state)
- Pros: Flexibility to choose best tool
- Cons: Inconsistent, harder to maintain, confusing for new contributors

**Suggested Decision Point**: After completing Phase 2 of documentation generation, evaluate which pattern makes documentation clearest and most maintainable. Document recommendation in `ARCHITECTURE.md` and update `CONTRIBUTING.md` with coding standards.

## Proposed Solution

### Approach: Python AST-Based Documentation Generator

Create a Python script that:
1. Parses all route files using AST (Abstract Syntax Tree)
2. Extracts route metadata (URL, methods, docstrings, parameters)
3. Analyzes related models and services for data structures
4. Generates comprehensive markdown documentation
5. Can be run manually or as part of CI/CD

### Why This Approach?

**Pros**:
- ✅ Native Python parsing - no external dependencies for core functionality
- ✅ Accurate extraction - works directly with Python AST
- ✅ Flexible formatting - can generate any markdown structure
- ✅ Incremental adoption - can start simple and add features
- ✅ Type hint support - can extract parameter types if added
- ✅ Docstring parsing - uses existing documentation patterns

**Cons**:
- ⚠️ Requires consistent docstring format
- ⚠️ May need manual annotations for complex cases
- ⚠️ Request/response schemas need to be inferred or documented

### Alternative Approaches Considered

1. **Flask-RESTX / Flask-Swagger**
   - Pros: Industry standard, generates interactive docs
   - Cons: Requires significant code refactoring, adds decorators everywhere

2. **OpenAPI Manual Specification**
   - Pros: Complete control, standard format
   - Cons: Completely manual, prone to drift from code

3. **Sphinx + AutoAPI**
   - Pros: Professional documentation site
   - Cons: Overkill for API docs, complex setup

## Implementation Plan

### Phase 1: Core Parser (Week 1)

**Goal**: Extract basic route information

**Script**: `backend/scripts/generate_api_docs.py`

**Functionality**:
```python
# Extract from each route file:
- Route URL pattern
- HTTP methods (GET, POST, PUT, DELETE)
- Function name
- Docstring (full text)
- Route decorators (@track_request, etc.)
- Parameter names from URL patterns
```

**Output**: Basic markdown with route table

**Example Output**:
```markdown
## IBKR Routes

### POST /api/ibkr/inbox/<transaction_id>/allocate

Allocate transaction to portfolios.

**Parameters**:
- `transaction_id` (path, required) - Transaction UUID

**Methods**: POST
```

### Phase 2: Enhanced Parsing (Week 2)

**Goal**: Parse docstrings into structured format

**Enhancements**:
```python
# Parse docstring sections:
- Summary (first line)
- Description (paragraph)
- Request body schema (if documented)
- Response schema (if documented)
- Args section
- Returns section
- Raises section (error handling)
```

**Docstring Convention** (recommend adopting):
```python
def endpoint():
    """
    One-line summary.

    Longer description paragraph.

    Request:
        {
            "field": "type - description"
        }

    Response:
        {
            "field": "type - description"
        }

    Args:
        param_name: Description

    Returns:
        Description of return value

    Raises:
        ErrorType: When this error occurs
    """
```

**Output**: Structured markdown with sections

### Phase 3: Model Integration (Week 3)

**Goal**: Link to data models for request/response schemas

**Enhancements**:
```python
# Analyze models.py to extract:
- SQLAlchemy model fields
- Field types
- Field constraints (nullable, unique, etc.)
- Relationships
- Generate JSON schema examples

# Link endpoints to models:
- Detect model references in code
- Auto-generate request/response examples
```

**Example**:
```python
# From model:
class IBKRTransaction(db.Model):
    id = db.Column(db.String(36), primary_key=True)
    symbol = db.Column(db.String(50), nullable=False)
    transaction_type = db.Column(db.String(20), nullable=False)
    total_amount = db.Column(db.Float, nullable=False)

# Generate documentation:
{
    "id": "uuid-string",
    "symbol": "AAPL",
    "transaction_type": "buy",
    "total_amount": 2500.00
}
```

### Phase 4: Advanced Features (Week 4)

**Goal**: Add authentication, errors, examples

**Enhancements**:
```python
# Authentication detection:
- Document if route requires auth
- Extract from decorators or middleware

# Error response catalog:
- Document common error responses
- HTTP status codes
- Error message formats

# Real examples:
- Option to include real examples from tests
- Sanitize sensitive data
```

### Phase 5: Integration & Automation (Week 5)

**Goal**: Make it part of development workflow

**Tasks**:
1. Add to pre-commit hooks (optional - only if changed)
2. Add to CI/CD pipeline
3. Create convenience commands:
   ```bash
   flask docs generate        # Generate docs
   flask docs validate        # Check for missing docs
   flask docs serve          # Serve interactive docs (future)
   ```
4. Add to CONTRIBUTING.md guide

## Documentation Output Format

### Structure

```markdown
# Investment Portfolio Manager - API Reference

Auto-generated on: 2025-01-XX

## Table of Contents
- System Routes
- Portfolio Routes
- Fund Routes
- Transaction Routes
- Dividend Routes
- IBKR Routes
- Developer Routes

---

## System Routes

### GET /api/system/version

Get application version and feature flags.

**Authentication**: None required

**Parameters**: None

**Response**: 200 OK
```json
{
    "version": "1.3.1",
    "db_version": "1.3.1",
    "features": {
        "ibkr_integration": true
    }
}
```

**Errors**:
- 500: Server error

**Example**:
```bash
curl http://localhost:5001/api/system/version
```

---

## IBKR Routes

### POST /api/ibkr/inbox/bulk-allocate

Process multiple transactions with identical allocations.

**Authentication**: Required

**Request Body**:
```json
{
    "transaction_ids": ["uuid1", "uuid2"],
    "allocations": [
        {
            "portfolio_id": "uuid",
            "percentage": 50.0
        }
    ]
}
```

**Response**: 200 OK
```json
{
    "success": true,
    "processed": 8,
    "failed": 2,
    "errors": [...]
}
```

**Validation Rules**:
- Allocations must sum to 100% (±0.01 tolerance)
- Each transaction must be in 'pending' status
- All transactions must have matching funds in portfolios

**Errors**:
- 400: Invalid request (allocations don't sum to 100%)
- 404: Transaction not found
- 500: Server error

**Example**:
```bash
curl -X POST http://localhost:5001/api/ibkr/inbox/bulk-allocate \
  -H "Content-Type: application/json" \
  -d '{"transaction_ids": [...], "allocations": [...]}'
```

**Related**:
- GET /api/ibkr/inbox - List all transactions
- POST /api/ibkr/inbox/<id>/allocate - Single allocation
```

### Grouping Strategy

Routes grouped by:
1. **Primary feature** (IBKR, Portfolio, Fund, etc.)
2. **Resource hierarchy** (list → get → create → update → delete)
3. **Related operations** linked with "See also" sections

### File Output

- **Primary**: `docs/API_REFERENCE.md` - Complete reference
- **Optional**: Split by module (`docs/api/ibkr.md`, `docs/api/portfolios.md`, etc.)
- **Index**: Update `README.md` with link

## Script Design

### File Structure

```
backend/scripts/
├── generate_api_docs.py          # Main script
├── api_doc_generator/            # Module
│   ├── __init__.py
│   ├── parser.py                 # AST parsing
│   ├── docstring_parser.py       # Docstring extraction
│   ├── model_analyzer.py         # Model schema generation
│   ├── markdown_generator.py     # Output formatting
│   └── templates.py              # Markdown templates
```

### Main Script Interface

```python
#!/usr/bin/env python3
"""
Generate API documentation from Flask routes.

Usage:
    python generate_api_docs.py                    # Generate all docs
    python generate_api_docs.py --module ibkr      # Only IBKR routes
    python generate_api_docs.py --validate         # Check coverage
    python generate_api_docs.py --format json      # JSON output
"""

import argparse
from api_doc_generator import DocumentationGenerator

def main():
    parser = argparse.ArgumentParser(description='Generate API documentation')
    parser.add_argument('--module', help='Specific module to document')
    parser.add_argument('--validate', action='store_true', help='Validate documentation coverage')
    parser.add_argument('--format', choices=['markdown', 'json', 'openapi'], default='markdown')
    parser.add_argument('--output', help='Output file path', default='docs/API_REFERENCE.md')

    args = parser.parse_args()

    generator = DocumentationGenerator()

    if args.validate:
        generator.validate_coverage()
    else:
        generator.generate(
            module=args.module,
            format=args.format,
            output=args.output
        )

if __name__ == '__main__':
    main()
```

### Core Parser Logic

```python
# parser.py
import ast
import os
from pathlib import Path

class RouteParser:
    """Parse Flask route files and extract endpoint information."""

    def parse_route_file(self, filepath: str) -> list[dict]:
        """
        Parse a single route file.

        Returns:
            List of route dictionaries with metadata
        """
        with open(filepath, 'r') as f:
            tree = ast.parse(f.read())

        routes = []

        for node in ast.walk(tree):
            if isinstance(node, ast.FunctionDef):
                route_info = self._extract_route_info(node)
                if route_info:
                    routes.append(route_info)

        return routes

    def _extract_route_info(self, func_node: ast.FunctionDef) -> dict | None:
        """Extract route information from function decorators."""
        route_decorator = None

        for decorator in func_node.decorator_list:
            if self._is_route_decorator(decorator):
                route_decorator = decorator
                break

        if not route_decorator:
            return None

        return {
            'url': self._extract_url(route_decorator),
            'methods': self._extract_methods(route_decorator),
            'function': func_node.name,
            'docstring': ast.get_docstring(func_node),
            'parameters': [arg.arg for arg in func_node.args.args],
            'decorators': [self._get_decorator_name(d) for d in func_node.decorator_list]
        }

    def _is_route_decorator(self, decorator) -> bool:
        """Check if decorator is a route decorator."""
        if isinstance(decorator, ast.Call):
            if hasattr(decorator.func, 'attr') and decorator.func.attr == 'route':
                return True
        return False

    # Additional helper methods...
```

### Docstring Parser

```python
# docstring_parser.py
import re
from typing import Optional

class DocstringParser:
    """Parse structured docstrings into components."""

    def parse(self, docstring: str) -> dict:
        """
        Parse docstring into structured components.

        Returns:
            {
                'summary': str,
                'description': str,
                'request': Optional[str],
                'response': Optional[str],
                'args': dict[str, str],
                'returns': str,
                'raises': dict[str, str]
            }
        """
        if not docstring:
            return self._empty_doc()

        lines = docstring.strip().split('\n')

        return {
            'summary': self._extract_summary(lines),
            'description': self._extract_description(lines),
            'request': self._extract_section(lines, 'Request'),
            'response': self._extract_section(lines, 'Response'),
            'args': self._extract_args(lines),
            'returns': self._extract_returns(lines),
            'raises': self._extract_raises(lines),
        }

    def _extract_summary(self, lines: list[str]) -> str:
        """Extract first non-empty line as summary."""
        for line in lines:
            line = line.strip()
            if line:
                return line
        return "No description"

    # Additional parsing methods...
```

### Markdown Generator

```python
# markdown_generator.py
from typing import List, Dict

class MarkdownGenerator:
    """Generate markdown documentation from parsed route data."""

    def generate(self, routes: List[Dict], module_name: str) -> str:
        """Generate complete markdown for a module."""
        sections = [
            self._generate_header(module_name),
            self._generate_toc(routes),
            self._generate_routes(routes)
        ]

        return '\n\n'.join(sections)

    def _generate_route(self, route: Dict) -> str:
        """Generate markdown for single route."""
        template = f"""
### {' '.join(route['methods'])} {route['url']}

{route['docstring']['summary']}

{self._generate_authentication(route)}

{self._generate_parameters(route)}

{self._generate_request_body(route)}

{self._generate_response(route)}

{self._generate_errors(route)}

{self._generate_example(route)}
"""
        return template

    # Additional generation methods...
```

## Maintenance Plan

### Documentation Standards

**Required for All Routes**:
1. At minimum: One-line summary in docstring
2. For user-facing routes: Full docstring with Request/Response sections
3. For internal routes: Mark with `@internal` decorator or comment

**Docstring Template** (add to CONTRIBUTING.md):
```python
def my_endpoint():
    """
    One-line summary (required).

    Longer description explaining purpose, business logic,
    and when to use this endpoint (optional but recommended).

    Request:
        {
            "field_name": "type - description",
            ...
        }

    Response:
        {
            "field_name": "type - description",
            ...
        }

    Args:
        param_name: Description of parameter

    Returns:
        Description of return value structure

    Raises:
        HTTPException: When and why this error occurs
    """
```

### Review Process

1. **On PR**: Script runs and generates updated docs
2. **Manual Review**: Developer checks generated docs are accurate
3. **Commit**: Updated `API_REFERENCE.md` committed with code changes
4. **CI Check**: Ensure docs were regenerated if routes changed

### Coverage Tracking

Script should track and report:
- Total routes: X
- Documented routes: Y (Z%)
- Missing docstrings: [list]
- Incomplete docstrings: [list]

Target: 100% coverage for user-facing routes, 80%+ for internal routes

## Future Enhancements

### Phase 6+: Advanced Features

1. **Interactive Documentation**
   - Generate Swagger/OpenAPI spec
   - Use Swagger UI or ReDoc for browsing
   - Try-it-out functionality

2. **Request/Response Validation**
   - Add Pydantic models for validation
   - Use for both runtime validation and doc generation
   - Type safety + automatic docs

3. **Code Examples**
   - Generate client code examples (Python, JavaScript, cURL)
   - Multi-language support
   - Copy-paste ready snippets

4. **Version Tracking**
   - Document when endpoints were added
   - Track breaking changes
   - Version comparison

5. **Testing Integration**
   - Extract examples from integration tests
   - Validate examples still work
   - Documentation as tests

6. **Performance Metrics**
   - Document typical response times
   - Rate limits
   - Pagination details

## Success Metrics

### Quantitative
- ✅ 100% of user-facing routes documented
- ✅ Documentation generation time < 5 seconds
- ✅ Zero manual documentation updates for routes
- ✅ API docs updated on every route change

### Qualitative
- ✅ Developers find docs helpful and accurate
- ✅ New contributors can understand API structure
- ✅ Documentation doesn't drift from implementation
- ✅ Reduces questions in issues/discussions

## Timeline

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| Phase 1 | Week 1 | Basic route extraction + table |
| Phase 2 | Week 2 | Structured docstring parsing |
| Phase 3 | Week 3 | Model integration + examples |
| Phase 4 | Week 4 | Auth, errors, validation |
| Phase 5 | Week 5 | CI/CD integration + polish |
| **Total** | **5 weeks** | **Production-ready system** |

## Resources Required

- **Development Time**: ~40 hours (1 week full-time or 5 weeks part-time)
- **Dependencies**: None (uses standard library)
- **Testing**: Unit tests for parser, integration tests for output
- **Documentation**: This plan + CONTRIBUTING.md updates

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Inconsistent docstrings | Medium | Provide template, add linter check |
| Multiple route implementation patterns | Medium | Parser handles all patterns initially; evaluate standardization during Phase 3 |
| Complex route patterns | Low | Handle common patterns, document edge cases |
| Performance with large codebase | Low | Currently ~60 routes, script should be fast |
| Resistance to adoption | Medium | Make it easy, show value early, optional at first |
| Code refactor burden | High | Don't force immediate standardization; gradual migration over time |

## Next Steps

1. **Review & Approve Plan**: Discuss and finalize approach
2. **Create Branch**: `feature/api-doc-generator`
3. **Phase 1 Implementation**: Start with basic parser
4. **Early Feedback**: Generate initial docs, get feedback
5. **Iterate**: Add features based on feedback
6. **Integrate**: Add to development workflow

## Appendix: Example Usage

### Generating Documentation

```bash
# From backend directory
cd /path/to/Investment-Portfolio-Manager/backend

# Generate all API docs
python scripts/generate_api_docs.py

# Output: docs/API_REFERENCE.md created
# Contains: All routes from all modules
```

### Validating Coverage

```bash
# Check documentation coverage
python scripts/generate_api_docs.py --validate

# Output:
# API Documentation Coverage Report
# =================================
# Total Routes: 63
# Documented: 45 (71.4%)
# Missing Docstrings: 18
#
# Routes needing documentation:
# - POST /api/developer/logs/clear
# - GET /api/developer/exchange-rate
# ...
```

### Module-Specific Generation

```bash
# Generate docs only for IBKR routes
python scripts/generate_api_docs.py --module ibkr --output docs/api/ibkr.md
```

### JSON Output (for tooling)

```bash
# Generate JSON for other tools
python scripts/generate_api_docs.py --format json --output api.json
```

---

**Document Version**: 1.0
**Created**: 2025-01-XX
**Status**: Proposal - Awaiting Approval
**Next Review**: After Phase 1 completion
