# Data Models

## Core Models

### Portfolio
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| name | String(100) | Portfolio name |
| description | Text | Optional description |
| is_archived | Boolean | Archive status |
| exclude_from_overview | Boolean | Whether to exclude from overview |

#### Relationships
- `funds`: One-to-many with PortfolioFund
- `realized_gains_losses`: One-to-many with RealizedGainLoss

### Fund
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| name | String(100) | Fund name |
| isin | String(12) | International Securities ID |
| symbol | String(10) | Trading symbol (optional) |
| currency | String(3) | Trading currency code |
| exchange | String(50) | Exchange name |
| investment_type | Enum | FUND or STOCK |
| dividend_type | Enum | NONE, CASH, or STOCK |

#### Relationships
- `historical_prices`: One-to-many with FundPrice
- `portfolios`: One-to-many with PortfolioFund
- `dividends`: One-to-many with Dividend
- `realized_gains_losses`: One-to-many with RealizedGainLoss

### PortfolioFund
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| portfolio_id | String(36) | Foreign key to Portfolio |
| fund_id | String(36) | Foreign key to Fund |

#### Relationships
- `portfolio`: Many-to-one with Portfolio
- `fund`: Many-to-one with Fund
- `transactions`: One-to-many with Transaction

#### Constraints
- Unique constraint on (portfolio_id, fund_id)

### Transaction
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| portfolio_fund_id | String(36) | Foreign key to PortfolioFund |
| date | Date | Transaction date |
| type | String(10) | 'buy', 'sell', or 'dividend' |
| shares | Float | Number of shares |
| cost_per_share | Float | Cost per share |
| created_at | DateTime | Creation timestamp |

#### Relationships
- `portfolio_fund`: Many-to-one with PortfolioFund
- `realized_gain_loss`: One-to-one with RealizedGainLoss

#### Indexes
- `ix_transaction_date`
- `ix_transaction_portfolio_fund_id`
- `ix_transaction_portfolio_fund_id_date`

### RealizedGainLoss
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| portfolio_id | String(36) | Foreign key to Portfolio |
| fund_id | String(36) | Foreign key to Fund |
| transaction_id | String(36) | Foreign key to Transaction |
| transaction_date | Date | Date of sale |
| shares_sold | Float | Number of shares sold |
| cost_basis | Float | Original purchase cost |
| sale_proceeds | Float | Amount received from sale |
| realized_gain_loss | Float | Profit/loss amount |
| created_at | DateTime | Creation timestamp |

#### Relationships
- `portfolio`: Many-to-one with Portfolio
- `fund`: Many-to-one with Fund
- `transaction`: One-to-one with Transaction

#### Indexes
- `ix_realized_gain_loss_portfolio_id`
- `ix_realized_gain_loss_fund_id`
- `ix_realized_gain_loss_transaction_date`
- `ix_realized_gain_loss_transaction_id`

### FundPrice
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| fund_id | String(36) | Foreign key to Fund |
| date | Date | Price date |
| price | Float | Price value |

#### Indexes
- `ix_fund_price_date`
- `ix_fund_price_fund_id`
- `ix_fund_price_fund_id_date`

### Dividend
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| fund_id | String(36) | Foreign key to Fund |
| portfolio_fund_id | String(36) | Foreign key to PortfolioFund |
| record_date | Date | Dividend record date |
| ex_dividend_date | Date | Ex-dividend date |
| shares_owned | Float | Shares owned |
| dividend_per_share | Float | Dividend per share |
| total_amount | Float | Total dividend amount |
| reinvestment_status | Enum | PENDING, COMPLETED, PARTIAL |
| buy_order_date | Date | Reinvestment date (optional) |
| reinvestment_transaction_id | String(36) | Foreign key to Transaction |
| created_at | DateTime | Creation timestamp |

#### Indexes
- `ix_dividend_fund_id`
- `ix_dividend_portfolio_fund_id`
- `ix_dividend_record_date`

### System Models

#### SystemSetting
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| key | Enum | Setting key |
| value | String(255) | Setting value |
| updated_at | DateTime | Last update timestamp |

#### Log
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| timestamp | DateTime | Event timestamp |
| level | Enum | DEBUG, INFO, WARNING, ERROR, CRITICAL |
| category | Enum | PORTFOLIO, FUND, TRANSACTION, etc. |
| message | Text | Log message |
| details | Text | JSON-formatted details |
| source | String(255) | Event source |
| request_id | String(36) | Associated request ID |
| http_status | Integer | HTTP status code |

#### SymbolInfo
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| symbol | String(10) | Trading symbol |
| name | String(255) | Symbol name |
| exchange | String(50) | Exchange |
| currency | String(3) | Currency code |
| isin | String(12) | ISIN if available |
| last_updated | DateTime | Last update timestamp |
| data_source | String(50) | Data source identifier |
| is_valid | Boolean | Validity flag |

## IBKR Flex Integration Models (v1.3.0+)

### IBKRConfig
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| flex_token | String(500) | Encrypted IBKR Flex token |
| flex_query_id | String(100) | IBKR Flex query ID |
| token_expires_at | DateTime | Token expiration date (max 1 year) |
| last_import_date | DateTime | Last successful import timestamp |
| auto_import_enabled | Boolean | Whether automated imports are enabled |
| created_at | DateTime | Configuration creation timestamp |
| updated_at | DateTime | Configuration update timestamp |

#### Notes
- Only one configuration record exists (singleton)
- Token is encrypted using Fernet encryption
- Token is write-only in the UI
- Token expiration tracking provides 30-day warning
- IBKR tokens are valid for maximum 1 year

### IBKRTransaction
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| ibkr_transaction_id | String(100) | IBKR's unique transaction ID |
| transaction_date | Date | Transaction date |
| symbol | String(10) | Trading symbol |
| isin | String(12) | ISIN code |
| description | Text | Transaction description |
| transaction_type | String(20) | 'buy', 'sell', 'dividend', or 'fee' |
| quantity | Float | Number of shares (nullable for fees) |
| price | Float | Price per share (nullable for fees) |
| total_amount | Float | Total transaction amount |
| currency | String(3) | Currency code |
| fees | Float | Transaction fees |
| status | String(20) | 'pending', 'processed', or 'ignored' |
| imported_at | DateTime | Import timestamp |
| processed_at | DateTime | Processing timestamp |
| raw_data | Text | JSON of original IBKR data |

#### Relationships
- `allocations`: One-to-many with IBKRTransactionAllocation

#### Indexes
- `ix_ibkr_transaction_status`
- `ix_ibkr_transaction_date`
- `ix_ibkr_transaction_ibkr_id`

#### Notes
- `ibkr_transaction_id` is unique to prevent duplicates
- Kept as audit trail even after processing
- Status tracks workflow: pending → processed/ignored

### IBKRTransactionAllocation
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| ibkr_transaction_id | String(36) | Foreign key to IBKRTransaction |
| portfolio_id | String(36) | Foreign key to Portfolio |
| allocation_percentage | Float | Allocation percentage |
| allocated_amount | Float | Allocated amount in currency |
| allocated_shares | Float | Allocated shares |
| transaction_id | String(36) | Foreign key to Transaction |
| created_at | DateTime | Allocation creation timestamp |

#### Relationships
- `ibkr_transaction`: Many-to-one with IBKRTransaction
- `portfolio`: Many-to-one with Portfolio
- `transaction`: Many-to-one with Transaction

#### Indexes
- `ix_ibkr_allocation_ibkr_transaction_id`
- `ix_ibkr_allocation_portfolio_id`
- `ix_ibkr_allocation_transaction_id`

#### Notes
- Multiple allocations per IBKR transaction for splitting across portfolios
- Percentages across allocations must sum to 100%
- Links IBKR transactions to created Transaction records

### IBKRImportCache
| Field | Type | Description |
|-------|------|-------------|
| id | String(36) | Primary key (UUID) |
| cache_key | String(255) | Cache key (query_id + date) |
| data | Text | Cached API response data |
| created_at | DateTime | Cache creation timestamp |
| expires_at | DateTime | Cache expiration timestamp |

#### Indexes
- `ix_ibkr_cache_expires_at`

#### Notes
- Cache duration: 24 hours
- Automatically cleaned up on import
- Cache key format: `ibkr_flex_{query_id}_{YYYY-MM-DD}`
