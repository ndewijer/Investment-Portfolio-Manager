# API Documentation with Flask-RESTX - Implementation Plan

**Created**: 2025-11-22
**Target Version**: 1.4.0 or 1.5.0
**Estimated Time**: 2-3 weeks (incremental adoption)
**Approach**: Replace custom AST parser plan with industry-standard Flask-RESTX

---

## Overview

Instead of building a custom AST-based documentation generator (864 lines, high maintenance), we'll use **Flask-RESTX** to provide:
- ✅ Automatic Swagger/OpenAPI documentation
- ✅ Interactive Swagger UI at `/api/docs`
- ✅ Request/response validation
- ✅ Type-safe API models
- ✅ Minimal code changes (incremental migration)

---

## Why Flask-RESTX?

### Comparison: Custom Parser vs Flask-RESTX

| Feature | Custom AST Parser | Flask-RESTX |
|---------|-------------------|-------------|
| **Initial Development** | 40+ hours | 10-15 hours |
| **Maintenance** | High (custom code) | Low (library maintained) |
| **Documentation Format** | Static markdown | Interactive Swagger UI |
| **Request Validation** | Manual | Automatic |
| **Response Validation** | Manual | Automatic |
| **Try-it-out** | No | Yes (Swagger UI) |
| **OpenAPI Spec** | Would need Phase 6+ | Built-in |
| **Learning Curve** | Custom patterns | Industry standard |
| **Community Support** | None | Active |

### Why Flask-RESTX over Flask-Smorest?

- **Flask-RESTX**: Extension of Flask, works with existing Blueprints, Swagger UI built-in
- **Flask-Smorest**: More opinionated, requires more refactoring, better for greenfield projects

**Decision**: Flask-RESTX is the better fit for incremental adoption.

---

## Implementation Plan

### Phase 1: Setup & Proof of Concept (Week 1, Days 1-2)

**Goal**: Get Flask-RESTX running with one route module as proof of concept

#### 1.1 Install Dependencies
```bash
pip install flask-restx
```

Add to `requirements.txt`:
```
flask-restx==1.3.0
```

#### 1.2 Create API Instance
**File**: `backend/app/__init__.py`

```python
from flask_restx import Api

# Create API instance
api = Api(
    version='1.3.3',
    title='Investment Portfolio Manager API',
    description='RESTful API for managing investment portfolios',
    doc='/api/docs',  # Swagger UI location
    prefix='/api'
)

def create_app(config_name="default"):
    app = Flask(__name__)
    # ... existing setup ...

    # Initialize API
    api.init_app(app)

    # Register namespaces (blueprints)
    from app.routes.system_routes import system_ns
    api.add_namespace(system_ns)

    return app
```

#### 1.3 Convert System Routes (Proof of Concept)
**File**: `backend/app/routes/system_routes.py`

**Before**:
```python
from flask import Blueprint
system = Blueprint("system", __name__)

@system.route("/system/version", methods=["GET"])
def get_version():
    """Get application version."""
    return jsonify({"version": "1.3.3"})
```

**After**:
```python
from flask_restx import Namespace, Resource, fields

system_ns = Namespace('system', description='System health and version operations')

# Define response model
version_model = system_ns.model('Version', {
    'version': fields.String(required=True, description='Application version'),
    'db_version': fields.String(required=True, description='Database schema version'),
    'features': fields.Raw(description='Feature flags')
})

@system_ns.route('/version')
class VersionAPI(Resource):
    """Version information endpoint."""

    @system_ns.doc('get_version')
    @system_ns.marshal_with(version_model)
    def get(self):
        """
        Get application version and feature flags.

        Returns version information including app version, database version,
        and enabled features like IBKR integration.
        """
        version_info = SystemService.get_version_info()
        return version_info, 200
```

**Deliverable**: Swagger UI accessible at `http://localhost:5001/api/docs` showing system routes

---

### Phase 2: Convert Portfolio Routes (Week 1, Days 3-5)

**Goal**: Convert portfolio CRUD operations with full request/response models

#### 2.1 Define Models
**File**: `backend/app/routes/portfolio_routes.py`

```python
from flask_restx import Namespace, Resource, fields

portfolio_ns = Namespace('portfolios', description='Portfolio management operations')

# Request models
create_portfolio_model = portfolio_ns.model('CreatePortfolio', {
    'name': fields.String(required=True, description='Portfolio name', min_length=1, max_length=100),
    'description': fields.String(description='Portfolio description', max_length=500),
    'currency': fields.String(required=True, description='Portfolio currency (EUR, USD, etc.)', pattern='^[A-Z]{3}$'),
    'exclude_from_overview': fields.Boolean(default=False, description='Exclude from overview page'),
})

update_portfolio_model = portfolio_ns.model('UpdatePortfolio', {
    'name': fields.String(description='Portfolio name', min_length=1, max_length=100),
    'description': fields.String(description='Portfolio description', max_length=500),
    'currency': fields.String(description='Portfolio currency', pattern='^[A-Z]{3}$'),
    'exclude_from_overview': fields.Boolean(description='Exclude from overview page'),
})

# Response models
portfolio_summary_model = portfolio_ns.model('PortfolioSummary', {
    'id': fields.String(required=True, description='Portfolio UUID'),
    'name': fields.String(required=True, description='Portfolio name'),
    'description': fields.String(description='Portfolio description'),
    'currency': fields.String(required=True, description='Portfolio currency'),
    'is_archived': fields.Boolean(required=True, description='Archive status'),
    'exclude_from_overview': fields.Boolean(required=True),
    'total_value': fields.Float(description='Total portfolio value'),
    'total_invested': fields.Float(description='Total amount invested'),
    'total_gain_loss': fields.Float(description='Total gain/loss'),
})

portfolio_detail_model = portfolio_ns.model('PortfolioDetail', {
    'id': fields.String(required=True),
    'name': fields.String(required=True),
    'description': fields.String(),
    'currency': fields.String(required=True),
    'is_archived': fields.Boolean(required=True),
    'exclude_from_overview': fields.Boolean(required=True),
    'funds': fields.List(fields.Nested(portfolio_ns.model('PortfolioFund', {
        'fund_id': fields.String(),
        'fund_name': fields.String(),
        'shares': fields.Float(),
        'current_value': fields.Float(),
        'invested_amount': fields.Float(),
        'gain_loss': fields.Float(),
    }))),
    'total_value': fields.Float(),
    'total_invested': fields.Float(),
    'total_gain_loss': fields.Float(),
})
```

#### 2.2 Convert Routes to Resources
```python
@portfolio_ns.route('/')
class PortfolioListAPI(Resource):
    """Portfolio list operations."""

    @portfolio_ns.doc('list_portfolios')
    @portfolio_ns.marshal_list_with(portfolio_summary_model)
    def get(self):
        """
        Get all portfolios.

        Returns a list of all portfolios with summary information including
        current value, total invested, and gain/loss.
        """
        portfolios = PortfolioService.get_portfolios_list(
            include_archived=request.args.get('include_archived', 'false').lower() == 'true'
        )
        return portfolios, 200

    @portfolio_ns.doc('create_portfolio')
    @portfolio_ns.expect(create_portfolio_model, validate=True)
    @portfolio_ns.marshal_with(portfolio_detail_model, code=201)
    @portfolio_ns.response(400, 'Validation Error')
    def post(self):
        """
        Create a new portfolio.

        Creates a new portfolio with the specified name, currency, and optional description.
        The portfolio will be empty initially - add funds using the portfolio-funds endpoint.
        """
        try:
            portfolio = PortfolioService.create_portfolio(request.json)
            return PortfolioService.format_portfolio_detail(portfolio), 201
        except ValueError as e:
            portfolio_ns.abort(400, str(e))


@portfolio_ns.route('/<string:portfolio_id>')
@portfolio_ns.param('portfolio_id', 'Portfolio UUID')
class PortfolioAPI(Resource):
    """Single portfolio operations."""

    @portfolio_ns.doc('get_portfolio')
    @portfolio_ns.marshal_with(portfolio_detail_model)
    @portfolio_ns.response(404, 'Portfolio not found')
    def get(self, portfolio_id):
        """
        Get portfolio details.

        Returns detailed information about a specific portfolio including
        all funds, positions, and financial metrics.
        """
        try:
            portfolio = PortfolioService.get_portfolio_detail(portfolio_id)
            return portfolio, 200
        except ValueError as e:
            portfolio_ns.abort(404, str(e))

    @portfolio_ns.doc('update_portfolio')
    @portfolio_ns.expect(update_portfolio_model, validate=True)
    @portfolio_ns.marshal_with(portfolio_detail_model)
    @portfolio_ns.response(404, 'Portfolio not found')
    def put(self, portfolio_id):
        """Update portfolio information."""
        try:
            portfolio = PortfolioService.update_portfolio(portfolio_id, request.json)
            return PortfolioService.format_portfolio_detail(portfolio), 200
        except ValueError as e:
            portfolio_ns.abort(404, str(e))

    @portfolio_ns.doc('delete_portfolio')
    @portfolio_ns.response(204, 'Portfolio deleted successfully')
    @portfolio_ns.response(404, 'Portfolio not found')
    @portfolio_ns.response(400, 'Portfolio cannot be deleted (has funds)')
    def delete(self, portfolio_id):
        """
        Delete a portfolio.

        Deletes a portfolio. The portfolio must be empty (no funds) before it can be deleted.
        Use the archive endpoint if you want to hide a portfolio without deleting it.
        """
        try:
            PortfolioService.delete_portfolio(portfolio_id)
            return '', 204
        except ValueError as e:
            if "not found" in str(e).lower():
                portfolio_ns.abort(404, str(e))
            else:
                portfolio_ns.abort(400, str(e))
```

**Deliverable**: Portfolio routes fully documented in Swagger UI with request/response validation

---

### Phase 3: Convert Remaining Routes (Week 2)

Convert in priority order:

#### Day 1-2: Transaction & Dividend Routes
- **transaction_ns**: CRUD for transactions, realized gain/loss tracking
- **dividend_ns**: CRUD for dividends, reinvestment handling

#### Day 3-4: Fund Routes
- **fund_ns**: CRUD for funds, symbol lookup, price updates

#### Day 5: IBKR Routes
- **ibkr_ns**: Complex allocation logic, bulk operations, dividend matching

---

### Phase 4: Advanced Features (Week 3)

#### 4.1 Request Validation
Already handled by `@expect(model, validate=True)`

Benefits:
- Automatic 400 errors for invalid requests
- Type checking
- Required field validation
- Pattern matching (regex)
- Min/max length validation

#### 4.2 Response Models with Nested Data
```python
transaction_model = portfolio_ns.model('Transaction', {
    'id': fields.String(required=True),
    'date': fields.Date(required=True),
    'type': fields.String(required=True, enum=['buy', 'sell', 'dividend', 'fee']),
    'shares': fields.Float(required=True),
    'cost_per_share': fields.Float(required=True),
    'total_value': fields.Float(required=True),
    'fund': fields.Nested(portfolio_ns.model('FundBasic', {
        'id': fields.String(),
        'name': fields.String(),
        'symbol': fields.String(),
    })),
    'realized_gain_loss': fields.Float(description='For sell transactions only'),
})
```

#### 4.3 Error Models
```python
error_model = api.model('Error', {
    'message': fields.String(required=True, description='Error message'),
    'errors': fields.Raw(description='Detailed validation errors'),
})

# Use in responses
@portfolio_ns.response(400, 'Bad Request', error_model)
@portfolio_ns.response(500, 'Internal Server Error', error_model)
```

#### 4.4 Authentication Documentation
```python
# Define security (if/when added)
authorizations = {
    'apikey': {
        'type': 'apiKey',
        'in': 'header',
        'name': 'X-API-Key'
    }
}

api = Api(
    # ... existing config ...
    authorizations=authorizations,
    security='apikey'
)

# Mark routes that require auth
@portfolio_ns.doc(security='apikey')
def get(self):
    """Protected endpoint."""
    pass
```

---

## Migration Strategy

### Incremental Adoption (Recommended)

**Week 1**: System + Portfolio routes
**Week 2**: Transaction, Dividend, Fund routes
**Week 3**: IBKR routes + polish

**Benefits**:
- Test each namespace independently
- Minimal risk (can keep old routes until new ones verified)
- Team can learn incrementally
- Can deploy partial migrations

### Coexistence Pattern
```python
# Old blueprint (deprecated)
@portfolios.route("/portfolios", methods=["GET"])
def get_portfolios_old():
    """DEPRECATED: Use /api/portfolios instead."""
    # ... keep for backwards compatibility ...

# New RESTX resource
@portfolio_ns.route('/')
class PortfolioListAPI(Resource):
    def get(self):
        """Get all portfolios."""
        # ... new implementation ...
```

**Deprecation Timeline**:
1. v1.4.0: Add Flask-RESTX alongside existing routes
2. v1.4.1: Mark old routes as deprecated in docs
3. v1.5.0: Remove old routes entirely

---

## Swagger UI Customization

### Custom UI Theme
```python
from flask_restx import Api

api = Api(
    # ... existing config ...
    doc='/api/docs',
    # Custom Swagger UI configuration
    ui=True,  # Enable Swagger UI
    validate=True,  # Enable request validation
)

# Custom CSS (optional)
@app.route('/swagger-ui-custom.css')
def swagger_ui_css():
    return """
    .swagger-ui .topbar { display: none }
    .swagger-ui .info .title { color: #2c3e50; }
    """
```

### Organize by Tags
```python
@portfolio_ns.route('/')
class PortfolioListAPI(Resource):
    @portfolio_ns.doc('list_portfolios', tags=['Portfolio Management'])
    def get(self):
        """Get all portfolios."""
        pass
```

---

## Testing Strategy

### Test Request Validation
```python
def test_create_portfolio_missing_name(client):
    """Test that missing required field returns 400."""
    response = client.post('/api/portfolios/', json={
        "currency": "EUR"
        # Missing required 'name' field
    })

    assert response.status_code == 400
    data = response.get_json()
    assert 'name' in data['errors']  # Flask-RESTX validation error
```

### Test Response Models
```python
def test_get_portfolio_response_structure(client, sample_portfolio):
    """Test that response matches defined model."""
    response = client.get(f'/api/portfolios/{sample_portfolio.id}')

    assert response.status_code == 200
    data = response.get_json()

    # Verify response structure matches model
    assert 'id' in data
    assert 'name' in data
    assert 'total_value' in data
    assert 'funds' in data
    assert isinstance(data['funds'], list)
```

### Test Swagger UI Endpoint
```python
def test_swagger_ui_accessible(client):
    """Test that Swagger UI is accessible."""
    response = client.get('/api/docs')
    assert response.status_code == 200
    assert b'swagger-ui' in response.data

def test_openapi_spec(client):
    """Test that OpenAPI spec is generated."""
    response = client.get('/api/swagger.json')
    assert response.status_code == 200

    spec = response.get_json()
    assert spec['info']['title'] == 'Investment Portfolio Manager API'
    assert 'paths' in spec
    assert '/portfolios/' in spec['paths']
```

---

## Documentation Quality

### Docstring Guidelines
```python
@portfolio_ns.route('/<string:portfolio_id>')
class PortfolioAPI(Resource):
    """Single portfolio operations."""

    @portfolio_ns.doc('get_portfolio')
    @portfolio_ns.marshal_with(portfolio_detail_model)
    @portfolio_ns.response(404, 'Portfolio not found')
    def get(self, portfolio_id):
        """
        Get portfolio details.

        Returns detailed information about a specific portfolio including:
        - All funds in the portfolio
        - Current positions (shares owned)
        - Financial metrics (total value, invested amount, gain/loss)
        - Historical performance data

        The response includes nested fund information for each holding.
        """
        # implementation...
```

**Best Practices**:
1. **Summary line**: One-line description (shows in Swagger UI list)
2. **Detailed description**: Explain purpose, behavior, edge cases
3. **Parameters**: Documented via `@param` decorator
4. **Responses**: Document all status codes with `@response`
5. **Examples**: Include in docstring if helpful

---

## Comparison to Old Plan

| Aspect | Custom AST Parser | Flask-RESTX |
|--------|-------------------|-------------|
| **Lines of Code** | ~2000+ (generator + templates) | ~500 (models + routes) |
| **Maintenance** | Custom code to maintain | Library maintained |
| **Time to Implement** | 5 weeks (40 hours) | 2-3 weeks (15-20 hours) |
| **Documentation Output** | Static markdown | Interactive Swagger UI |
| **Request Validation** | Manual in routes | Automatic |
| **Type Safety** | None | Model-based |
| **Try-It-Out** | No | Yes (Swagger UI) |
| **OpenAPI Spec** | Phase 6+ (future) | Built-in |
| **Learning Curve** | Custom patterns | Industry standard |
| **Community Support** | None | Active community |
| **CI/CD Integration** | Custom script in CI | Standard library |

---

## Success Metrics

### Quantitative
- ✅ 100% of routes documented in Swagger UI
- ✅ All request/response models defined
- ✅ Interactive API docs accessible at `/api/docs`
- ✅ Request validation prevents invalid data
- ✅ OpenAPI spec auto-generated

### Qualitative
- ✅ Developers can explore API without reading code
- ✅ New contributors understand API structure quickly
- ✅ Frontend developers can test endpoints interactively
- ✅ API documentation never drifts from implementation
- ✅ Request/response validation catches bugs early

---

## Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|-------------|
| Phase 1 | Days 1-2 | Flask-RESTX setup + system routes |
| Phase 2 | Days 3-5 | Portfolio routes converted |
| Phase 3 | Week 2 | All remaining routes converted |
| Phase 4 | Week 3 | Advanced features + polish |
| **Total** | **2-3 weeks** | **Production-ready API docs** |

**Time Savings vs Custom Parser**: 2-3 weeks saved (vs 5 weeks for custom solution)

---

## Resources Required

- **Development Time**: 15-20 hours (vs 40 hours for custom parser)
- **Dependencies**: `flask-restx==1.3.0` (single package)
- **Testing**: Standard route tests + model validation tests
- **Documentation**: Update CONTRIBUTING.md with API model patterns

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Learning curve for Flask-RESTX | Low | Well-documented library, similar to Flask patterns |
| Migration complexity | Medium | Incremental adoption, keep old routes during migration |
| Breaking changes | Low | Add new routes alongside old ones, deprecate gradually |
| Model maintenance | Low | Models live next to routes, easy to update together |

---

## Next Steps

1. **Approve Plan**: Review and approve Flask-RESTX approach
2. **Spike**: Quick 2-hour spike to validate approach with system routes
3. **Phase 1**: Full implementation of setup + system routes
4. **Iterate**: Get feedback, then proceed with remaining phases
5. **Document**: Add Flask-RESTX patterns to CONTRIBUTING.md

---

## Appendix: Example Swagger UI Output

When complete, visiting `http://localhost:5001/api/docs` will show:

```
Investment Portfolio Manager API v1.3.3

Namespaces:
├── system - System health and version operations
│   ├── GET /system/version - Get application version
│   └── GET /system/health - Check system health
│
├── portfolios - Portfolio management operations
│   ├── GET /portfolios/ - List all portfolios
│   ├── POST /portfolios/ - Create new portfolio
│   ├── GET /portfolios/{portfolio_id} - Get portfolio details
│   ├── PUT /portfolios/{portfolio_id} - Update portfolio
│   └── DELETE /portfolios/{portfolio_id} - Delete portfolio
│
├── funds - Fund management operations
│   ├── GET /funds/ - List all funds
│   ├── POST /funds/ - Create new fund
│   └── ... (etc)
│
└── ... (all other namespaces)
```

Each endpoint shows:
- **Parameters**: With types, validation rules, examples
- **Request Body**: JSON schema with examples
- **Responses**: All status codes with example responses
- **Try It Out**: Button to test endpoint interactively
- **Models**: Expandable schemas showing all fields

---

**Document Version**: 1.0
**Created**: 2025-11-22
**Status**: Proposal - Ready for Implementation
**Replaces**: API_DOCUMENTATION_GENERATION_PLAN.md (custom AST parser approach)
