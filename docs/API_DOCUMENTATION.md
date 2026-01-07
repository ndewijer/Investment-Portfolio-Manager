# API Documentation

This document provides comprehensive information about the Investment Portfolio Manager REST API, implemented using Flask-RESTX with automatic Swagger/OpenAPI documentation.

---

## Overview

The Investment Portfolio Manager provides a comprehensive REST API for managing investment portfolios, funds, transactions, and Interactive Brokers (IBKR) integration. The API is built with Flask-RESTX and includes automatic Swagger UI documentation.

### Key Features

- **Automatic Documentation**: Interactive Swagger UI at `/api/docs`
- **Type-Safe Schemas**: Request/response models with validation
- **Comprehensive Coverage**: 72 endpoints across 7 namespaces
- **Service Layer Architecture**: Business logic separated from HTTP interface
- **Error Handling**: Standardized error responses with appropriate HTTP status codes
- **Logging**: Comprehensive request/response logging for debugging

---

## Accessing the API

### Swagger UI

The interactive API documentation is available at:

```
http://localhost:5001/api/docs
```

The Swagger UI provides:
- Complete endpoint documentation
- Request/response schemas
- Interactive testing interface
- Example values for all parameters
- HTTP status code descriptions

### Base URL

All API endpoints are prefixed with `/api`:

```
http://localhost:5001/api/{namespace}/{endpoint}
```

---

## API Namespaces

The API is organized into 7 logical namespaces:

### 1. System (`/api/system`)

System health and version information.

**Endpoints**: 2
- `GET /api/system/health` - Health check
- `GET /api/system/version` - Application version and database status

**Use Cases**:
- Monitoring system availability
- Checking version compatibility
- Detecting pending database migrations

### 2. Portfolio (`/api/portfolio`)

Portfolio management and analytics.

**Endpoints**: 13
- CRUD operations for portfolios
- Portfolio summary and history
- Fund-portfolio relationships
- Archive/unarchive portfolios
- Portfolio fund value history

**Use Cases**:
- Creating and managing investment portfolios
- Tracking portfolio performance over time
- Adding/removing funds from portfolios
- Archiving inactive portfolios

### 3. Fund (`/api/fund`)

Investment fund and stock management.

**Endpoints**: 13
- CRUD operations for funds
- Price history and updates
- Symbol information lookup
- Fund usage checking
- Bulk price updates

**Use Cases**:
- Adding new funds or stocks
- Updating current and historical prices
- Looking up fund information by symbol
- Checking if a fund can be safely deleted

### 4. Transaction (`/api/transactions`)

Transaction recording and management.

**Endpoints**: 5
- CRUD operations for transactions
- Buy, sell, dividend, and fee transactions
- Automatic realized gain/loss calculation

**Use Cases**:
- Recording buy/sell transactions
- Tracking dividend payments
- Calculating realized gains and losses
- Managing transaction history

### 5. Dividend (`/api/dividends`)

Dividend tracking and reinvestment.

**Endpoints**: 6
- CRUD operations for dividends
- Cash and stock dividend support
- Reinvestment tracking
- Fund and portfolio dividend queries

**Use Cases**:
- Recording dividend payments
- Tracking stock dividend reinvestment
- Querying dividend history by fund or portfolio

### 6. IBKR (`/api/ibkr`)

Interactive Brokers integration.

**Endpoints**: 19
- Configuration management
- Transaction import via Flex Query
- Transaction inbox management
- Portfolio allocation
- Dividend matching
- Bulk operations

**Use Cases**:
- Importing transactions from Interactive Brokers
- Allocating imported transactions to portfolios
- Managing unallocated transactions
- Matching IBKR dividends to existing records

### 7. Developer (`/api/developer`)

Development and debugging utilities with comprehensive CSV import capabilities.

**Endpoints**: 15
- System logs viewing and clearing
- Logging configuration management
- Exchange rate management
- Fund price management
- CSV template generation
- CSV transaction and fund price imports
- Database introspection utilities

**Key Features**:
- **CSV Import Validation**: Centralized UTF-8 encoding and header validation
- **Logging Management**: View, clear, and configure system logging settings
- **Manual Data Entry**: Override exchange rates and fund prices for testing
- **Developer Tools**: Access system logs and database information

**Use Cases**:
- Debugging system issues through log analysis
- Importing transaction and fund price data from CSV files
- Manually setting exchange rates or fund prices for development/testing
- Configuring system logging levels and behavior
- Downloading properly formatted CSV import templates
- Bulk importing historical data from external sources

---

## Common Request Patterns

### Creating Resources

All creation endpoints follow a similar pattern:

**Request**:
```http
POST /api/{namespace}
Content-Type: application/json

{
  "field1": "value1",
  "field2": "value2"
}
```

**Response** (201 Created):
```json
{
  "id": "uuid-here",
  "field1": "value1",
  "field2": "value2",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Retrieving Resources

**Single Resource**:
```http
GET /api/{namespace}/{id}
```

**Collection**:
```http
GET /api/{namespace}
```

**With Filters**:
```http
GET /api/{namespace}?filter1=value1&filter2=value2
```

### Updating Resources

```http
PUT /api/{namespace}/{id}
Content-Type: application/json

{
  "field1": "new_value"
}
```

### Deleting Resources

```http
DELETE /api/{namespace}/{id}
```

---

## Request/Response Models

All endpoints use typed request and response models defined in Flask-RESTX. These models:

- **Validate Input**: Ensure required fields are present and correctly typed
- **Document API**: Automatically appear in Swagger UI
- **Type Safety**: Prevent common API integration errors

### Example Model Definition

```python
portfolio_model = ns.model('Portfolio', {
    'id': fields.String(required=True, description='Portfolio UUID'),
    'name': fields.String(required=True, description='Portfolio name'),
    'description': fields.String(description='Portfolio description'),
    'exclude_from_overview': fields.Boolean(description='Exclude from overview'),
    'is_archived': fields.Boolean(description='Archive status'),
    'created_at': fields.DateTime(description='Creation timestamp')
})
```

---

## Error Handling

The API uses standard HTTP status codes and returns consistent error responses:

### HTTP Status Codes

- `200 OK` - Successful GET, PUT, or operation
- `201 Created` - Successful POST (resource created)
- `204 No Content` - Successful DELETE
- `400 Bad Request` - Invalid input or validation error
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict (e.g., fund in use, confirmation required)
- `500 Internal Server Error` - Server-side error

### Error Response Format

```json
{
  "error": "Brief error description",
  "details": "Detailed error information (optional)"
}
```

### Example Error Responses

**Validation Error (400)**:
```json
{
  "error": "Missing required fields: name and description"
}
```

**Not Found (404)**:
```json
{
  "error": "Portfolio not found"
}
```

**Conflict (409)**:
```json
{
  "error": "Cannot delete fund while attached to 3 portfolios",
  "details": {
    "portfolios": ["Portfolio A", "Portfolio B", "Portfolio C"]
  }
}
```

---

## Authentication & Security

### Current Implementation

The API currently implements:

- **API Key Protection**: Certain endpoints (e.g., bulk price updates) require API key authentication
- **IBKR Token Encryption**: Flex API tokens are encrypted at rest
- **Input Validation**: All inputs validated via Flask-RESTX models

### API Key Usage

Endpoints marked with `security='apikey'` require an API key:

```http
GET /api/fund/update-all-prices
X-API-Key: your-api-key-here
```

Configure the API key in `.env`:
```
API_KEY=your-secure-api-key
```

---

## Common Use Cases

### Use Case 1: Creating a New Portfolio

**Goal**: Create a portfolio and add a fund to it

**Steps**:

1. **Create Portfolio**:
```http
POST /api/portfolio
Content-Type: application/json

{
  "name": "My Growth Portfolio",
  "description": "Long-term growth investments"
}
```

Response: `{ "id": "portfolio-uuid", ... }`

2. **Create or Find Fund**:
```http
GET /api/fund/symbol/AAPL
```

Response: `{ "symbol": "AAPL", "name": "Apple Inc.", ... }`

```http
POST /api/fund
Content-Type: application/json

{
  "name": "Apple Inc.",
  "isin": "US0378331005",
  "symbol": "AAPL",
  "currency": "USD",
  "exchange": "NASDAQ",
  "investment_type": "stock",
  "dividend_type": "cash"
}
```

Response: `{ "id": "fund-uuid", ... }`

3. **Add Fund to Portfolio**:
```http
POST /api/portfolio/funds
Content-Type: application/json

{
  "portfolio_id": "portfolio-uuid",
  "fund_id": "fund-uuid"
}
```

### Use Case 2: Recording a Stock Purchase

**Goal**: Record buying 100 shares of AAPL at $150

**Steps**:

1. **Get Portfolio-Fund Relationship ID**:
```http
GET /api/portfolio/funds?portfolio_id=portfolio-uuid
```

Response: `[{ "id": "portfolio-fund-uuid", ... }]`

2. **Create Transaction**:
```http
POST /api/transactions
Content-Type: application/json

{
  "portfolio_fund_id": "portfolio-fund-uuid",
  "date": "2024-01-15",
  "type": "buy",
  "shares": 100,
  "cost_per_share": 150.00
}
```

### Use Case 3: Importing from Interactive Brokers

**Goal**: Import and allocate IBKR transactions

**Steps**:

1. **Configure IBKR**:
```http
POST /api/ibkr/config
Content-Type: application/json

{
  "flex_token": "your-ibkr-token",
  "flex_query_id": "123456",
  "auto_import_enabled": false
}
```

2. **Test Connection** (Optional):
```http
POST /api/ibkr/config/test
Content-Type: application/json

{
  "flex_token": "your-ibkr-token",
  "flex_query_id": "123456"
}
```

3. **Import Transactions**:
```http
POST /api/ibkr/import
```

Response: `{ "imported": 5, "skipped": 2, ... }`

4. **View Pending Transactions**:
```http
GET /api/ibkr/inbox?status=pending
```

5. **Allocate Transaction to Portfolios**:
```http
POST /api/ibkr/inbox/{transaction-id}/allocate
Content-Type: application/json

{
  "allocations": [
    { "portfolio_id": "portfolio-1-uuid", "percentage": 60.0 },
    { "portfolio_id": "portfolio-2-uuid", "percentage": 40.0 }
  ]
}
```

---

## Migration from Legacy API

### Coexistence

The Swagger API coexists with the legacy Blueprint routes during the transition period:

- **Legacy Routes**: Continue to work at their original paths (e.g., `/portfolios`)
- **Swagger Routes**: Available with `/api` prefix (e.g., `/api/portfolio`)

### Key Differences

| Aspect | Legacy API | Swagger API |
|--------|-----------|-------------|
| **Base Path** | `/` | `/api/` |
| **Documentation** | Manual/none | Automatic Swagger UI |
| **Validation** | Manual | Automatic via models |
| **Type Safety** | Limited | Comprehensive |
| **Error Responses** | Varied | Standardized |

### Compatibility Notes

1. **Response Format**: Response formats are consistent across all Swagger API endpoints

2. **Business Logic**: All APIs use the same service layer, ensuring consistent behavior

---

## Query Parameters

Many endpoints support optional query parameters for filtering and pagination:

### Common Parameters

- `start_date` - Filter by start date (YYYY-MM-DD)
- `end_date` - Filter by end date (YYYY-MM-DD)
- `portfolio_id` - Filter by portfolio UUID
- `fund_id` - Filter by fund UUID
- `status` - Filter by status (pending, processed, ignored)
- `type` - Filter by type (buy, sell, dividend, fee)

### Example

```http
GET /api/transactions?portfolio_id=uuid&start_date=2024-01-01&end_date=2024-12-31&type=buy
```

---

## Date Formats

All dates use ISO 8601 format:

- **Date**: `YYYY-MM-DD` (e.g., `2024-01-15`)
- **DateTime**: `YYYY-MM-DDTHH:MM:SSZ` (e.g., `2024-01-15T10:30:00Z`)

**Note**: Always use UTC for timestamps.

---

## Pagination

Currently, the API does not implement pagination. All collection endpoints return complete result sets.

**Future Enhancement**: Pagination will be added in a future version for large result sets.

---

## Rate Limiting

The API does not currently implement rate limiting.

**Note**: The `/api/fund/update-all-prices` endpoint requires an API key and should be called sparingly (typically once per day) to avoid overwhelming external price data providers.

---

## Best Practices

### 1. Use UUIDs for Resource IDs

All resources use UUIDs as identifiers. Always store the complete UUID:

```python
# Good
portfolio_id = "f47ac10b-58cc-4372-a567-0e02b2c3d479"

# Bad
portfolio_id = "f47ac10b"  # Truncated
```

### 2. Handle Errors Gracefully

Always check HTTP status codes and handle errors:

```javascript
try {
  const response = await fetch('/api/portfolio', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(portfolioData)
  });

  if (!response.ok) {
    const error = await response.json();
    console.error('API Error:', error.error);
    return;
  }

  const portfolio = await response.json();
  // Success handling
} catch (error) {
  console.error('Network Error:', error);
}
```

### 3. Validate Input Before Sending

Use the Swagger UI or model definitions to understand required fields:

```javascript
// Always include required fields
const transaction = {
  portfolio_fund_id: "uuid",  // Required
  date: "2024-01-15",         // Required
  type: "buy",                // Required
  shares: 100,                // Required
  cost_per_share: 150.00      // Required
};
```

### 4. Use Appropriate HTTP Methods

- **GET** - Retrieve data (no side effects)
- **POST** - Create new resources
- **PUT** - Update existing resources
- **DELETE** - Remove resources

### 5. Check for Confirmations

Some destructive operations require confirmation:

```javascript
// First attempt may require confirmation
const response = await fetch(`/api/portfolio/fund/${id}`, {
  method: 'DELETE'
});

if (response.status === 409) {
  const data = await response.json();
  if (data.requires_confirmation) {
    // Show user the impact (transaction_count, dividend_count)
    // If confirmed:
    await fetch(`/api/portfolio/fund/${id}?confirm=true`, {
      method: 'DELETE'
    });
  }
}
```

---

## Testing the API

### Using Swagger UI

1. Navigate to `http://localhost:5001/api/docs`
2. Expand an endpoint
3. Click "Try it out"
4. Fill in parameters
5. Click "Execute"
6. View the response

### Using cURL

```bash
# GET request
curl http://localhost:5001/api/portfolio

# POST request
curl -X POST http://localhost:5001/api/portfolio \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Portfolio", "description": "Testing"}'

# With authentication
curl http://localhost:5001/api/fund/update-all-prices \
  -H "X-API-Key: your-api-key"
```

### Using Postman

1. Import the OpenAPI spec from `/api/docs`
2. Postman will create a complete collection
3. Set the base URL to `http://localhost:5001`
4. Start testing endpoints

---

## Troubleshooting

### Issue: "Resource not found" (404)

**Cause**: Invalid UUID or resource doesn't exist
**Solution**: Verify the UUID is correct and the resource exists

```bash
# List all portfolios to verify UUIDs
curl http://localhost:5001/api/portfolio
```

### Issue: "Validation error" (400)

**Cause**: Missing required fields or invalid data types
**Solution**: Check Swagger UI for required fields and types

### Issue: "Internal Server Error" (500)

**Cause**: Server-side error
**Solution**:
1. Check server logs: `docker compose logs backend`
2. Check database connectivity
3. Verify environment variables are set correctly

### Issue: Swagger UI Not Loading

**Cause**: Flask-RESTX not properly initialized
**Solution**:
1. Verify Flask-RESTx is installed: `pip list | grep Flask-RESTx`
2. Check `run.py` for proper API initialization
3. Restart the backend: `docker compose restart backend`

---

## Version History

### Version 1.3.3 - Swagger Implementation

- **Initial Release**: Complete Swagger/OpenAPI documentation
- **Coverage**: 72 endpoints across 7 namespaces
- **Features**: Interactive Swagger UI, typed models, error handling
- **Compatibility**: Coexists with legacy Blueprint routes

---

## Future Enhancements

Planned improvements for future versions:

1. **Pagination**: Add pagination for large result sets
2. **Rate Limiting**: Implement rate limiting for API protection
3. **Authentication**: Enhanced authentication options (OAuth, JWT)
4. **Webhooks**: Event notifications for portfolio changes
5. **GraphQL**: Alternative GraphQL endpoint for flexible queries
6. **API Versioning**: Explicit API versioning (v1, v2, etc.)

---

## Related Documentation

- **Architecture**: `docs/ARCHITECTURE.md` - System architecture overview
- **IBKR Integration**: `docs/IBKR_FEATURES.md` - Interactive Brokers integration details
- **Security**: `docs/SECURITY.md` - Security best practices
- **Testing**: `docs/TESTING.md` - API testing guide

---

## Support

### Reporting Issues

If you encounter API issues:

1. Check the Swagger UI for correct request format
2. Review server logs for errors
3. Create an issue on GitHub with:
   - Endpoint being called
   - Request payload
   - Expected vs actual response
   - Server logs (if available)

### Contributing

API improvements and additions are welcome! See `CONTRIBUTING.md` for guidelines.

---

**Last Updated**: 2024 (Version 1.3.3)
**Maintained By**: @ndewijer
