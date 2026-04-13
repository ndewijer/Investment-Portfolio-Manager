# API Documentation

This document provides comprehensive information about the Investment Portfolio Manager REST API, implemented as a Go backend using the Chi router.

---

## Overview

The Investment Portfolio Manager provides a comprehensive REST API for managing investment portfolios, funds, transactions, and Interactive Brokers (IBKR) integration. The API is built with Go using the [Chi](https://github.com/go-chi/chi) router and follows a Handler -> Service -> Repository layered architecture.

### Key Features

- **Layered Architecture**: Handler -> Service -> Repository separation of concerns
- **Type-Safe Request Parsing**: Strongly-typed Go request structs with validation
- **Comprehensive Coverage**: 70+ endpoints across 7 route groups
- **Middleware Stack**: Request ID injection, logging, CORS, panic recovery, UUID validation, API key authentication
- **Error Handling**: Standardized error responses with appropriate HTTP status codes
- **Structured Logging**: Per-request logging with request IDs for debugging and tracing

---

## Accessing the API

### Base URL

All API endpoints are prefixed with `/api`:

```
http://localhost:5000/api/{group}/{endpoint}
```

The Go backend listens on port `5000` by default (configurable via `SERVER_PORT` environment variable).

### Health Check

Verify the API is running:

```bash
curl http://localhost:5000/api/system/health
```

---

## API Route Groups

The API is organized into 7 logical route groups:

### 1. System (`/api/system`)

System health and version information.

**Endpoints**: 2

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/system/health` | Health check |
| GET | `/api/system/version` | Application version and database status |

**Use Cases**:
- Monitoring system availability
- Checking version compatibility
- Detecting pending database migrations

### 2. Portfolio (`/api/portfolio`)

Portfolio management and analytics.

**Endpoints**: 13

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/portfolio` | List all portfolios |
| POST | `/api/portfolio` | Create portfolio |
| GET | `/api/portfolio/{id}` | Get portfolio by ID |
| PUT | `/api/portfolio/{id}` | Update portfolio |
| DELETE | `/api/portfolio/{id}` | Delete portfolio |
| POST | `/api/portfolio/{id}/archive` | Archive portfolio |
| POST | `/api/portfolio/{id}/unarchive` | Unarchive portfolio |
| GET | `/api/portfolio/summary` | Portfolio summary (materialized) |
| GET | `/api/portfolio/history` | Portfolio history (materialized) |
| GET | `/api/portfolio/funds` | List all portfolio-fund relationships |
| GET | `/api/portfolio/funds/{id}` | Funds in a portfolio |
| POST | `/api/portfolio/funds` | Add fund to portfolio |
| DELETE | `/api/portfolio/fund/{id}` | Remove fund from portfolio |

**Use Cases**:
- Creating and managing investment portfolios
- Tracking portfolio performance over time
- Adding/removing funds from portfolios
- Archiving inactive portfolios

### 3. Fund (`/api/fund`)

Investment fund and stock management.

**Endpoints**: 11

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/fund` | List all funds |
| POST | `/api/fund` | Create fund |
| GET | `/api/fund/{id}` | Get fund details |
| PUT | `/api/fund/{id}` | Update fund |
| DELETE | `/api/fund/{id}` | Delete fund |
| GET | `/api/fund/{id}/check-usage` | Check if fund is in use |
| GET | `/api/fund/fund-prices/{id}` | Price history for a fund |
| POST | `/api/fund/fund-prices/{id}/update` | Update fund prices (Yahoo Finance) |
| GET | `/api/fund/history/{portfolioId}` | Historical fund values for portfolio |
| GET | `/api/fund/symbol/{symbol}` | Look up trading symbol |
| POST | `/api/fund/update-all-prices` | Update prices for all funds (API key required) |

**Use Cases**:
- Adding new funds or stocks
- Updating current and historical prices
- Looking up fund information by symbol
- Checking if a fund can be safely deleted

### 4. Transaction (`/api/transaction`)

Transaction recording and management.

**Endpoints**: 6

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/transaction` | List all transactions |
| POST | `/api/transaction` | Create transaction |
| GET | `/api/transaction/{id}` | Get transaction by ID |
| PUT | `/api/transaction/{id}` | Update transaction |
| DELETE | `/api/transaction/{id}` | Delete transaction |
| GET | `/api/transaction/portfolio/{id}` | Transactions for a portfolio |

**Use Cases**:
- Recording buy/sell transactions
- Tracking dividend payments
- Calculating realized gains and losses
- Managing transaction history

### 5. Dividend (`/api/dividend`)

Dividend tracking and reinvestment.

**Endpoints**: 7

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/dividend` | List all dividends |
| POST | `/api/dividend` | Create dividend |
| GET | `/api/dividend/{id}` | Get dividend by ID |
| PUT | `/api/dividend/{id}` | Update dividend |
| DELETE | `/api/dividend/{id}` | Delete dividend |
| GET | `/api/dividend/portfolio/{id}` | Dividends for a portfolio |
| GET | `/api/dividend/fund/{id}` | Dividends for a fund |

**Use Cases**:
- Recording dividend payments
- Tracking stock dividend reinvestment
- Querying dividend history by fund or portfolio

### 6. IBKR (`/api/ibkr`)

Interactive Brokers integration.

**Endpoints**: 19

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/ibkr/config` | Get IBKR configuration status |
| POST | `/api/ibkr/config` | Create or update IBKR configuration |
| DELETE | `/api/ibkr/config` | Delete IBKR configuration |
| POST | `/api/ibkr/config/test` | Test IBKR connection |
| POST | `/api/ibkr/import` | Trigger IBKR Flex report import |
| GET | `/api/ibkr/portfolios` | Available portfolios for allocation |
| GET | `/api/ibkr/dividend/pending` | Pending dividends for matching |
| GET | `/api/ibkr/inbox` | List imported IBKR transactions |
| GET | `/api/ibkr/inbox/count` | Count of IBKR inbox transactions |
| POST | `/api/ibkr/inbox/bulk-allocate` | Bulk allocate transactions |
| GET | `/api/ibkr/inbox/{id}` | Get IBKR transaction details |
| DELETE | `/api/ibkr/inbox/{id}` | Delete IBKR transaction |
| POST | `/api/ibkr/inbox/{id}/allocate` | Allocate transaction to portfolios |
| POST | `/api/ibkr/inbox/{id}/unallocate` | Unallocate a processed transaction |
| GET | `/api/ibkr/inbox/{id}/allocations` | Get allocation details |
| PUT | `/api/ibkr/inbox/{id}/allocations` | Modify allocation percentages |
| GET | `/api/ibkr/inbox/{id}/eligible-portfolios` | Get eligible portfolios for transaction |
| POST | `/api/ibkr/inbox/{id}/ignore` | Mark transaction as ignored |
| POST | `/api/ibkr/inbox/{id}/match-dividend` | Match dividend to existing records |

**Use Cases**:
- Importing transactions from Interactive Brokers
- Allocating imported transactions to portfolios
- Managing unallocated transactions
- Matching IBKR dividends to existing records

### 7. Developer (`/api/developer`)

Development and debugging utilities with comprehensive CSV import capabilities.

**Endpoints**: 13

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/developer/logs` | Get system logs (cursor-based) |
| GET | `/api/developer/logs/filter-options` | Distinct values for log filter picklists |
| DELETE | `/api/developer/logs` | Clear all system logs |
| GET | `/api/developer/system-settings/logging` | Get logging configuration |
| PUT | `/api/developer/system-settings/logging` | Update logging configuration |
| GET | `/api/developer/csv/fund-prices/template` | CSV template for fund price import |
| GET | `/api/developer/csv/transactions/template` | CSV template for transaction import |
| GET | `/api/developer/exchange-rate` | Get exchange rate for currency pair |
| POST | `/api/developer/exchange-rate` | Set exchange rate for currency pair |
| GET | `/api/developer/fund-price` | Get fund price for specific date |
| POST | `/api/developer/fund-price` | Set fund price for specific date |
| POST | `/api/developer/import-fund-prices` | Import fund prices from CSV |
| POST | `/api/developer/import-transactions` | Import transactions from CSV |

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
POST /api/{group}
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
GET /api/{group}/{id}
```

**Collection**:
```http
GET /api/{group}
```

**With Filters**:
```http
GET /api/{group}?filter1=value1&filter2=value2
```

### Updating Resources

```http
PUT /api/{group}/{id}
Content-Type: application/json

{
  "field1": "new_value"
}
```

### Deleting Resources

```http
DELETE /api/{group}/{id}
```

---

## Request/Response Models

All endpoints use strongly-typed Go request structs for parsing and validation. These are defined in the `internal/api/request/` package and ensure:

- **Validate Input**: Required fields are checked and types enforced at the handler level
- **Type Safety**: Go's type system prevents common API integration errors
- **Clear Contracts**: Request struct definitions serve as documentation for expected payloads

### Example Request Struct

```go
// internal/api/request/portfolio.go
type CreatePortfolioRequest struct {
    Name                string `json:"name"`
    Description         string `json:"description"`
    ExcludeFromOverview bool   `json:"exclude_from_overview"`
}
```

### Example Response Helper

```go
// internal/api/response/response.go
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func RespondError(w http.ResponseWriter, status int, message string) {
    RespondJSON(w, status, map[string]string{"error": message})
}
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

- **API Key Protection**: Certain endpoints (e.g., bulk price updates) require API key authentication via middleware
- **IBKR Token Encryption**: Flex API tokens are encrypted at rest using Fernet symmetric encryption
- **Input Validation**: All inputs validated via typed Go request structs and middleware (e.g., UUID path parameter validation)

### API Key Usage

Endpoints protected by the API key middleware require an `X-API-Key` header:

```http
POST /api/fund/update-all-prices
X-API-Key: your-api-key-here
```

Configure the API key in `.env`:
```
INTERNAL_API_KEY=your-secure-api-key
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
  "investmentType": "stock",
  "dividendType": "cash"
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
GET /api/portfolio/funds/portfolio-uuid
```

Response: `[{ "id": "portfolio-fund-uuid", ... }]`

2. **Create Transaction**:
```http
POST /api/transaction
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
  "flexToken": "your-ibkr-token",
  "flexQueryId": "123456",
  "autoImportEnabled": false,
  "enabled": true,
  "defaultAllocationEnabled": false,
  "defaultAllocations": [
    {
      "portfolioId": "uuid-here",
      "percentage": 50.0
    },
    {
      "portfolioId": "uuid-here-2",
      "percentage": 50.0
    }
  ]
}
```

2. **Test Connection** (Optional):
```http
POST /api/ibkr/config/test
Content-Type: application/json

{
  "flexToken": "your-ibkr-token",
  "flexQueryId": "123456"
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
GET /api/transaction?portfolio_id=uuid&start_date=2024-01-01&end_date=2024-12-31&type=buy
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

```go
// Good
portfolioID := "f47ac10b-58cc-4372-a567-0e02b2c3d479"

// Bad
portfolioID := "f47ac10b" // Truncated
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

Use the endpoint tables and request struct definitions to understand required fields:

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

### Using cURL

```bash
# GET request
curl http://localhost:5000/api/portfolio

# POST request
curl -X POST http://localhost:5000/api/portfolio \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Portfolio", "description": "Testing"}'

# With authentication
curl -X POST http://localhost:5000/api/fund/update-all-prices \
  -H "X-API-Key: your-api-key"
```

### Using Postman

1. Set the base URL to `http://localhost:5000`
2. Create requests using the endpoint tables above
3. Set `Content-Type: application/json` for POST/PUT requests
4. Start testing endpoints

---

## Troubleshooting

### Issue: "Resource not found" (404)

**Cause**: Invalid UUID or resource doesn't exist
**Solution**: Verify the UUID is correct and the resource exists

```bash
# List all portfolios to verify UUIDs
curl http://localhost:5000/api/portfolio
```

### Issue: "Validation error" (400)

**Cause**: Missing required fields or invalid data types
**Solution**: Check the request struct definitions in `internal/api/request/` for required fields and types

### Issue: "Internal Server Error" (500)

**Cause**: Server-side error
**Solution**:
1. Check server logs: `docker compose logs backend`
2. Check database connectivity
3. Verify environment variables are set correctly

### Issue: API Not Responding

**Cause**: Server not started or port conflict
**Solution**:
1. Verify the server is running: `docker compose ps`
2. Check port availability: `lsof -i :5000`
3. Restart the backend: `docker compose restart backend`

---

## Version History

### Version 1.3.3 - Initial API Documentation

- **Initial Release**: Complete API documentation
- **Coverage**: 72 endpoints across 7 namespaces
- **Features**: Typed models, error handling

### Version 2.0.0 - Go Backend Migration

- **Backend Rewrite**: Migrated from Python/Flask to Go/Chi
- **Same API Contract**: All endpoints maintain the same request/response format
- **Middleware Stack**: Request ID injection, logging, CORS, panic recovery, UUID validation
- **Performance**: Compiled binary with lower memory footprint

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
- **Configuration**: `docs/CONFIGURATION.md` - Environment variables and configuration
- **IBKR Integration**: `docs/IBKR_FEATURES.md` - Interactive Brokers integration details
- **Security**: `docs/SECURITY.md` - Security best practices
- **Testing**: `docs/TESTING.md` - API testing guide

---

## Support

### Reporting Issues

If you encounter API issues:

1. Check the endpoint tables above for correct request format
2. Review server logs for errors
3. Create an issue on GitHub with:
   - Endpoint being called
   - Request payload
   - Expected vs actual response
   - Server logs (if available)

### Contributing

API improvements and additions are welcome! See `CONTRIBUTING.md` for guidelines.

---

**Last Updated**: 2026-04-13 (Version 2.0.0)
**Maintained By**: @ndewijer
