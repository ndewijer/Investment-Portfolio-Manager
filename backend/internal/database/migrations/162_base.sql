-- +goose Up

-- Portfolio table
CREATE TABLE IF NOT EXISTS portfolio (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_archived BOOLEAN,
    exclude_from_overview BOOLEAN DEFAULT FALSE NOT NULL
);

-- Fund table
CREATE TABLE IF NOT EXISTS fund (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    isin VARCHAR(12) NOT NULL UNIQUE,
    symbol VARCHAR(10),
    currency VARCHAR(3) NOT NULL,
    exchange VARCHAR(50) NOT NULL,
    investment_type VARCHAR(5) NOT NULL,
    dividend_type VARCHAR(5) NOT NULL
);

-- Portfolio-Fund junction table
CREATE TABLE IF NOT EXISTS portfolio_fund (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    portfolio_id VARCHAR(36) NOT NULL,
    fund_id VARCHAR(36) NOT NULL,
    FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
    FOREIGN KEY(fund_id) REFERENCES fund(id) ON DELETE CASCADE,
    CONSTRAINT unique_portfolio_fund UNIQUE (portfolio_id, fund_id)
);

-- Transaction table (quoted because transaction is a reserved keyword)
CREATE TABLE IF NOT EXISTS "transaction" (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    portfolio_fund_id VARCHAR(36) NOT NULL,
    date DATE NOT NULL,
    type VARCHAR(10) NOT NULL,
    shares FLOAT NOT NULL,
    cost_per_share FLOAT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE
);

-- Dividend table
CREATE TABLE IF NOT EXISTS dividend (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    fund_id VARCHAR(36) NOT NULL,
    portfolio_fund_id VARCHAR(36) NOT NULL,
    record_date DATE NOT NULL,
    ex_dividend_date DATE NOT NULL,
    shares_owned FLOAT NOT NULL,
    dividend_per_share FLOAT NOT NULL,
    total_amount FLOAT NOT NULL,
    reinvestment_status VARCHAR(9) NOT NULL,
    buy_order_date DATE,
    reinvestment_transaction_id VARCHAR(36),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(fund_id) REFERENCES fund(id),
    FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
    FOREIGN KEY(reinvestment_transaction_id) REFERENCES "transaction"(id) ON DELETE RESTRICT
);

-- Fund price table
CREATE TABLE IF NOT EXISTS fund_price (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    fund_id VARCHAR(36) NOT NULL,
    date DATE NOT NULL,
    price FLOAT NOT NULL,
    FOREIGN KEY(fund_id) REFERENCES fund(id),
    CONSTRAINT unique_fund_price UNIQUE (fund_id, date)
);

-- Realized Gain/Loss table
CREATE TABLE IF NOT EXISTS realized_gain_loss (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    portfolio_id VARCHAR(36) NOT NULL,
    fund_id VARCHAR(36) NOT NULL,
    transaction_id VARCHAR(36) NOT NULL,
    transaction_date DATE NOT NULL,
    shares_sold FLOAT NOT NULL,
    cost_basis FLOAT NOT NULL,
    sale_proceeds FLOAT NOT NULL,
    realized_gain_loss FLOAT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
    FOREIGN KEY(fund_id) REFERENCES fund(id),
    FOREIGN KEY(transaction_id) REFERENCES "transaction"(id) ON DELETE CASCADE
);

-- Exchange rate table
CREATE TABLE IF NOT EXISTS exchange_rate (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    from_currency VARCHAR(3) NOT NULL,
    to_currency VARCHAR(3) NOT NULL,
    rate FLOAT NOT NULL,
    date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_exchange_rate UNIQUE (from_currency, to_currency, date)
);

-- System setting table
CREATE TABLE IF NOT EXISTS system_setting (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    "key" VARCHAR(15) NOT NULL UNIQUE,
    value VARCHAR(255) NOT NULL,
    updated_at DATETIME
);

-- Default system settings
-- https://vic.demuzere.be/articles/generating-uuids-in-sqlite/
INSERT OR IGNORE INTO system_setting (id, "key", value, updated_at)
VALUES ((select CONCAT(
  HEX(RANDOMBLOB(4)),
  '-',
  HEX(RANDOMBLOB(2)),
  '-',
  '4',
  SUBSTR(HEX(RANDOMBLOB(2)),0,4),
  '-',
  FORMAT('%X', 8 + ABS(RANDOM() % 4)),
  SUBSTR(HEX(RANDOMBLOB(2)),0,4),
  '-',
  SUBSTR(HEX(RANDOMBLOB(8)),0,13)
)), "LOGGING_ENABLED", "true", DateTime('now')),
((select CONCAT(
  HEX(RANDOMBLOB(4)),
  '-',
  HEX(RANDOMBLOB(2)),
  '-',
  '4',
  SUBSTR(HEX(RANDOMBLOB(2)),0,4),
  '-',
  FORMAT('%X', 8 + ABS(RANDOM() % 4)),
  SUBSTR(HEX(RANDOMBLOB(2)),0,4),
  '-',
  SUBSTR(HEX(RANDOMBLOB(8)),0,13)
)), "LOGGING_LEVEL", "info", DateTime('now'));

-- Symbol info table
CREATE TABLE IF NOT EXISTS symbol_info (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    exchange VARCHAR(50),
    currency VARCHAR(3),
    isin VARCHAR(12) UNIQUE,
    last_updated DATETIME,
    data_source VARCHAR(50),
    is_valid BOOLEAN
);

-- IBKR configuration table
CREATE TABLE IF NOT EXISTS ibkr_config (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    flex_token VARCHAR(500) NOT NULL,
    flex_query_id VARCHAR(100) NOT NULL, -- model uses int, but SQLite type affinity handles the coercion transparently
    token_expires_at DATETIME,
    last_import_date DATETIME,
    auto_import_enabled BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT (CURRENT_TIMESTAMP),
    updated_at DATETIME DEFAULT (CURRENT_TIMESTAMP),
    enabled BOOLEAN NOT NULL,
    default_allocation_enabled BOOLEAN NOT NULL,
    default_allocations TEXT
);

-- IBKR transaction table
CREATE TABLE IF NOT EXISTS "ibkr_transaction" (
    id VARCHAR(36) NOT NULL,
    ibkr_transaction_id VARCHAR(100) NOT NULL,
    transaction_date DATE NOT NULL,
    symbol VARCHAR(10),
    isin VARCHAR(12),
    description TEXT,
    transaction_type VARCHAR(20) NOT NULL,
    quantity FLOAT,
    price FLOAT,
    total_amount FLOAT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    fees FLOAT NOT NULL,
    status VARCHAR(20) NOT NULL,
    imported_at DATETIME DEFAULT (CURRENT_TIMESTAMP),
    processed_at DATETIME,
    raw_data TEXT,
    report_date DATE NOT NULL,
    notes VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (ibkr_transaction_id)
);

-- IBKR transaction allocation table
CREATE TABLE IF NOT EXISTS ibkr_transaction_allocation (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    ibkr_transaction_id VARCHAR(36) NOT NULL,
    portfolio_id VARCHAR(36) NOT NULL,
    allocation_percentage FLOAT NOT NULL,
    allocated_amount FLOAT NOT NULL,
    allocated_shares FLOAT NOT NULL,
    transaction_id VARCHAR(36),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(ibkr_transaction_id) REFERENCES ibkr_transaction(id) ON DELETE CASCADE,
    FOREIGN KEY(portfolio_id) REFERENCES portfolio(id) ON DELETE CASCADE,
    FOREIGN KEY(transaction_id) REFERENCES "transaction"(id) ON DELETE CASCADE
);

-- IBKR import cache table
CREATE TABLE IF NOT EXISTS ibkr_import_cache (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    cache_key VARCHAR(255) NOT NULL UNIQUE,
    data TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

-- Fund history materialized table
CREATE TABLE IF NOT EXISTS fund_history_materialized (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    portfolio_fund_id VARCHAR(36) NOT NULL,
    fund_id VARCHAR(36) NOT NULL,
    date VARCHAR(10) NOT NULL,
    shares FLOAT NOT NULL,
    price FLOAT NOT NULL,
    value FLOAT NOT NULL,
    cost FLOAT NOT NULL,
    realized_gain FLOAT NOT NULL,
    unrealized_gain FLOAT NOT NULL,
    total_gain_loss FLOAT NOT NULL,
    dividends FLOAT NOT NULL,
    fees FLOAT NOT NULL,
    calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY(portfolio_fund_id) REFERENCES portfolio_fund(id) ON DELETE CASCADE,
    CONSTRAINT uq_portfolio_fund_date UNIQUE (portfolio_fund_id, date)
);

-- Log table
CREATE TABLE IF NOT EXISTS log (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    level VARCHAR(8) NOT NULL,
    category VARCHAR(11) NOT NULL,
    message TEXT NOT NULL,
    details TEXT,
    source VARCHAR(255) NOT NULL,
    user_id VARCHAR(36),
    request_id VARCHAR(36),
    stack_trace TEXT,
    http_status INTEGER,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_portfolio_id ON realized_gain_loss(portfolio_id);
CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_fund_id ON realized_gain_loss(fund_id);
CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_transaction_date ON realized_gain_loss(transaction_date);
CREATE INDEX IF NOT EXISTS ix_realized_gain_loss_transaction_id ON realized_gain_loss(transaction_id);
CREATE INDEX IF NOT EXISTS ix_exchange_rate_date ON exchange_rate(date);
CREATE INDEX IF NOT EXISTS ix_fund_price_date ON fund_price(date);
CREATE INDEX IF NOT EXISTS ix_fund_price_fund_id ON fund_price(fund_id);
CREATE INDEX IF NOT EXISTS ix_fund_price_fund_id_date ON fund_price(fund_id, date);
CREATE INDEX IF NOT EXISTS ix_transaction_date ON "transaction"(date);
CREATE INDEX IF NOT EXISTS ix_transaction_portfolio_fund_id ON "transaction"(portfolio_fund_id);
CREATE INDEX IF NOT EXISTS ix_transaction_portfolio_fund_id_date ON "transaction"(portfolio_fund_id, date);
CREATE INDEX IF NOT EXISTS ix_ibkr_transaction_status ON ibkr_transaction(status);
CREATE INDEX IF NOT EXISTS ix_ibkr_transaction_date ON ibkr_transaction(transaction_date);
CREATE INDEX IF NOT EXISTS ix_ibkr_transaction_ibkr_id ON ibkr_transaction(ibkr_transaction_id);
CREATE INDEX IF NOT EXISTS ix_ibkr_allocation_ibkr_transaction_id ON ibkr_transaction_allocation(ibkr_transaction_id);
CREATE INDEX IF NOT EXISTS ix_ibkr_allocation_portfolio_id ON ibkr_transaction_allocation(portfolio_id);
CREATE INDEX IF NOT EXISTS ix_ibkr_allocation_transaction_id ON ibkr_transaction_allocation(transaction_id);
CREATE INDEX IF NOT EXISTS ix_ibkr_cache_expires_at ON ibkr_import_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_fund_history_pf_date ON fund_history_materialized(portfolio_fund_id, date);
CREATE INDEX IF NOT EXISTS idx_fund_history_date ON fund_history_materialized(date);
CREATE INDEX IF NOT EXISTS idx_fund_history_fund_id ON fund_history_materialized(fund_id);
CREATE INDEX IF NOT EXISTS ix_dividend_fund_id ON dividend(fund_id);
CREATE INDEX IF NOT EXISTS ix_dividend_portfolio_fund_id ON dividend(portfolio_fund_id);
CREATE INDEX IF NOT EXISTS ix_dividend_record_date ON dividend(record_date);
CREATE INDEX IF NOT EXISTS ix_log_timestamp_id ON log(timestamp, id);
CREATE INDEX IF NOT EXISTS ix_log_level ON log(level);
CREATE INDEX IF NOT EXISTS ix_log_category ON log(category);

-- +goose Down

DROP INDEX IF EXISTS ix_log_category;
DROP INDEX IF EXISTS ix_log_level;
DROP INDEX IF EXISTS ix_log_timestamp_id;
DROP INDEX IF EXISTS ix_dividend_record_date;
DROP INDEX IF EXISTS ix_dividend_portfolio_fund_id;
DROP INDEX IF EXISTS ix_dividend_fund_id;
DROP INDEX IF EXISTS idx_fund_history_fund_id;
DROP INDEX IF EXISTS idx_fund_history_date;
DROP INDEX IF EXISTS idx_fund_history_pf_date;
DROP INDEX IF EXISTS ix_ibkr_cache_expires_at;
DROP INDEX IF EXISTS ix_ibkr_allocation_transaction_id;
DROP INDEX IF EXISTS ix_ibkr_allocation_portfolio_id;
DROP INDEX IF EXISTS ix_ibkr_allocation_ibkr_transaction_id;
DROP INDEX IF EXISTS ix_ibkr_transaction_ibkr_id;
DROP INDEX IF EXISTS ix_ibkr_transaction_date;
DROP INDEX IF EXISTS ix_ibkr_transaction_status;
DROP INDEX IF EXISTS ix_transaction_portfolio_fund_id_date;
DROP INDEX IF EXISTS ix_transaction_portfolio_fund_id;
DROP INDEX IF EXISTS ix_transaction_date;
DROP INDEX IF EXISTS ix_fund_price_fund_id_date;
DROP INDEX IF EXISTS ix_fund_price_fund_id;
DROP INDEX IF EXISTS ix_fund_price_date;
DROP INDEX IF EXISTS ix_exchange_rate_date;
DROP INDEX IF EXISTS ix_realized_gain_loss_transaction_id;
DROP INDEX IF EXISTS ix_realized_gain_loss_transaction_date;
DROP INDEX IF EXISTS ix_realized_gain_loss_fund_id;
DROP INDEX IF EXISTS ix_realized_gain_loss_portfolio_id;

DROP TABLE IF EXISTS log;
DROP TABLE IF EXISTS fund_history_materialized;
DROP TABLE IF EXISTS ibkr_import_cache;
DROP TABLE IF EXISTS ibkr_transaction_allocation;
DROP TABLE IF EXISTS "ibkr_transaction";
DROP TABLE IF EXISTS ibkr_config;
DROP TABLE IF EXISTS symbol_info;
DROP TABLE IF EXISTS system_setting;
DROP TABLE IF EXISTS exchange_rate;
DROP TABLE IF EXISTS realized_gain_loss;
DROP TABLE IF EXISTS fund_price;
DROP TABLE IF EXISTS dividend;
DROP TABLE IF EXISTS "transaction";
DROP TABLE IF EXISTS portfolio_fund;
DROP TABLE IF EXISTS fund;
DROP TABLE IF EXISTS portfolio;
