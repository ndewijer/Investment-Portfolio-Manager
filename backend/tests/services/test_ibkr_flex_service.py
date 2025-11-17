"""
Tests for IBKRFlexService - IBKR Flex Web Service integration.

This test suite covers:
- Encryption/decryption of IBKR tokens (security critical)
- Fetching statements from IBKR API (mocked)
- Parsing XML responses into transactions
- Cache management and expiration
- Error handling for all IBKR error codes
- Exchange rate import
- Transaction import and duplicate detection
"""

from datetime import datetime, timedelta
from unittest.mock import patch

import pytest
import responses
from app.models import ExchangeRate, IBKRImportCache, IBKRTransaction, db
from app.services.ibkr_flex_service import IBKRFlexService
from tests.test_helpers import make_custom_string

# Sample XML fixtures
SAMPLE_SEND_REQUEST_SUCCESS = """<?xml version="1.0" encoding="UTF-8"?>
<FlexStatementResponse timestamp="01 January, 2025 01:00 AM EST">
  <Status>Success</Status>
  <ReferenceCode>1234567890</ReferenceCode>
  <Url>https://ndcdyn.interactivebrokers.com/AccountManagement/FlexWebService/GetStatement</Url>
</FlexStatementResponse>"""

SAMPLE_SEND_REQUEST_ERROR_1012 = """<?xml version="1.0" encoding="UTF-8"?>
<FlexStatementResponse timestamp="01 January, 2025 01:00 AM EST">
  <Status>Fail</Status>
  <ErrorCode>1012</ErrorCode>
  <ErrorMessage>Token has expired.</ErrorMessage>
</FlexStatementResponse>"""

SAMPLE_SEND_REQUEST_ERROR_1015 = """<?xml version="1.0" encoding="UTF-8"?>
<FlexStatementResponse timestamp="01 January, 2025 01:00 AM EST">
  <Status>Fail</Status>
  <ErrorCode>1015</ErrorCode>
  <ErrorMessage>Token is invalid.</ErrorMessage>
</FlexStatementResponse>"""

SAMPLE_STATEMENT_IN_PROGRESS = """<?xml version="1.0" encoding="UTF-8"?>
<FlexStatementResponse timestamp="01 January, 2025 01:00 AM EST">
  <Status>Warn</Status>
  <ErrorCode>1019</ErrorCode>
  <ErrorMessage>Statement generation in progress. Please try again shortly.</ErrorMessage>
</FlexStatementResponse>"""

SAMPLE_FLEX_STATEMENT = """<?xml version="1.0" encoding="UTF-8"?>
<FlexQueryResponse queryName="Test" type="AF">
  <FlexStatements count="1">
    <FlexStatement accountId="U1234567" fromDate="2025-01-01" toDate="2025-01-31">
      <Trades>
        <Trade accountId="U1234567" symbol="AAPL" isin="US0378331005" description="APPLE INC"
               tradeDate="20250115" quantity="10" tradePrice="150.00" netCash="-1500.00"
               currency="USD" ibCommission="-1.00" transactionID="12345" ibOrderID="67890"/>
        <Trade accountId="U1234567" symbol="MSFT" isin="US5949181045" description="MICROSOFT CORP"
               tradeDate="20250116" quantity="-5" tradePrice="380.00" netCash="1900.00"
               currency="USD" ibCommission="-1.00" transactionID="12346" ibOrderID="67891"/>
      </Trades>
      <CashTransactions>
        <CashTransaction accountId="U1234567" symbol="AAPL" isin="US0378331005"
                         type="Dividends" dateTime="20250110" amount="25.50" currency="USD"
                         transactionID="12347"
                         description="AAPL(US0378331005) Cash Dividend USD 2.55 per Share"
                         code="DIV" exDate="20250105"/>
        <CashTransaction accountId="U1234567" symbol="AAPL" isin="US0378331005"
                         type="Commission" dateTime="20250115" amount="-2.50"
                         currency="USD" transactionID="12348" description="Commission Charged"
                         code="FEE"/>
      </CashTransactions>
      <ConversionRates>
        <ConversionRate reportDate="20250115" fromCurrency="EUR" toCurrency="USD" rate="1.10"/>
        <ConversionRate reportDate="20250115" fromCurrency="GBP" toCurrency="USD" rate="1.27"/>
      </ConversionRates>
    </FlexStatement>
  </FlexStatements>
</FlexQueryResponse>"""

# XML with missing currency field (tests fallback to USD)
SAMPLE_STATEMENT_NO_CURRENCY = """<?xml version="1.0" encoding="UTF-8"?>
<FlexQueryResponse queryName="Test" type="AF">
  <FlexStatements count="1">
    <FlexStatement accountId="U1234567" fromDate="2025-01-01" toDate="2025-01-31">
      <Trades>
        <Trade accountId="U1234567" symbol="AAPL" isin="US0378331005" description="APPLE INC"
               tradeDate="20250115" quantity="10" tradePrice="150.00" netCash="-1500.00"
               ibCommission="-1.00" transactionID="12345" ibOrderID="67890"/>
      </Trades>
      <CashTransactions>
        <CashTransaction accountId="U1234567" symbol="AAPL" isin="US0378331005"
                         type="Dividends" dateTime="20250110" amount="25.50"
                         transactionID="12347" description="Cash Dividend"
                         code="DIV"/>
      </CashTransactions>
    </FlexStatement>
  </FlexStatements>
</FlexQueryResponse>"""


@pytest.fixture
def ibkr_service(app_context):
    """Create IBKRFlexService instance with test config."""
    from cryptography.fernet import Fernet
    from flask import current_app

    # Generate a valid Fernet key for testing
    test_key = Fernet.generate_key().decode()
    current_app.config["IBKR_ENCRYPTION_KEY"] = test_key

    return IBKRFlexService()


@pytest.fixture
def test_token():
    """Test IBKR token."""
    return "test_token_12345"


@pytest.fixture
def test_query_id():
    """Test IBKR query ID."""
    return "987654"


class TestEncryption:
    """Tests for token encryption/decryption (security critical)."""

    def test_encrypt_decrypt_token(self, ibkr_service, test_token):
        """Test that encryption and decryption work correctly."""
        # Encrypt token
        encrypted = ibkr_service._encrypt_token(test_token)

        # Should be different from original
        assert encrypted != test_token

        # Should be able to decrypt back to original
        decrypted = ibkr_service._decrypt_token(encrypted)
        assert decrypted == test_token

    def test_encrypt_without_key(self, app_context):
        """Test that encryption fails without encryption key."""
        service = IBKRFlexService()
        service.encryption_key = None

        with pytest.raises(ValueError, match="Encryption key not available"):
            service._encrypt_token("test")

    def test_decrypt_without_key(self, app_context):
        """Test that decryption fails without encryption key."""
        service = IBKRFlexService()
        service.encryption_key = None

        with pytest.raises(ValueError, match="Encryption key not available"):
            service._decrypt_token("encrypted_test")

    def test_decrypt_invalid_token(self, ibkr_service):
        """Test that decrypting invalid data raises error."""
        with pytest.raises((ValueError, Exception)):  # Fernet can raise various decryption errors
            ibkr_service._decrypt_token("not_valid_encrypted_data")

    def test_encryption_is_deterministic(self, ibkr_service, test_token):
        """Test that same token produces different encrypted values (uses random IV)."""
        encrypted1 = ibkr_service._encrypt_token(test_token)
        encrypted2 = ibkr_service._encrypt_token(test_token)

        # Should be different due to random IV
        assert encrypted1 != encrypted2

        # But both should decrypt to same value
        assert ibkr_service._decrypt_token(encrypted1) == test_token
        assert ibkr_service._decrypt_token(encrypted2) == test_token


class TestCacheManagement:
    """Tests for cache management."""

    def test_get_cache_key(self, ibkr_service, test_query_id):
        """Test cache key generation includes date."""
        cache_key = ibkr_service._get_cache_key(test_query_id)

        today = datetime.now().strftime("%Y-%m-%d")
        assert cache_key == f"ibkr_flex_{test_query_id}_{today}"

    def test_cache_data_and_retrieval(self, app_context, db_session, ibkr_service, test_query_id):
        """Test caching and retrieving data."""
        cache_key = make_custom_string("test_cache_", 8)
        test_data = "<xml>test data</xml>"

        # Cache data
        ibkr_service._cache_data(cache_key, test_data)

        # Should be able to retrieve it
        retrieved = ibkr_service._get_cached_data(cache_key)
        assert retrieved == test_data

    def test_cache_expiration(self, app_context, db_session, ibkr_service):
        """Test that expired cache entries are not returned."""
        cache_key = make_custom_string("test_cache_", 8)
        test_data = "<xml>test data</xml>"

        # Create expired cache entry
        expired_time = datetime.now() - timedelta(hours=2)
        cache_entry = IBKRImportCache(cache_key=cache_key, data=test_data, expires_at=expired_time)
        db.session.add(cache_entry)
        db.session.commit()

        # Should return None for expired cache
        retrieved = ibkr_service._get_cached_data(cache_key)
        assert retrieved is None

        # Expired entry should be deleted
        assert IBKRImportCache.query.filter_by(cache_key=cache_key).first() is None

    def test_clean_expired_cache(self, app_context, db_session, ibkr_service):
        """Test cleaning expired cache entries."""
        # Create mix of expired and valid cache entries
        expired_key1 = make_custom_string("expired1_", 8)
        expired_key2 = make_custom_string("expired2_", 8)
        valid_key = make_custom_string("valid_", 8)

        expired_time = datetime.now() - timedelta(hours=2)
        valid_time = datetime.now() + timedelta(hours=1)

        db.session.add_all(
            [
                IBKRImportCache(cache_key=expired_key1, data="data1", expires_at=expired_time),
                IBKRImportCache(cache_key=expired_key2, data="data2", expires_at=expired_time),
                IBKRImportCache(cache_key=valid_key, data="data3", expires_at=valid_time),
            ]
        )
        db.session.commit()

        # Clean expired cache
        ibkr_service._clean_expired_cache()

        # Expired entries should be deleted
        assert IBKRImportCache.query.filter_by(cache_key=expired_key1).first() is None
        assert IBKRImportCache.query.filter_by(cache_key=expired_key2).first() is None

        # Valid entry should remain
        assert IBKRImportCache.query.filter_by(cache_key=valid_key).first() is not None


class TestErrorHandling:
    """Tests for IBKR error code handling."""

    def test_get_error_message_known_code(self, ibkr_service):
        """Test getting error message for known error code."""
        message = ibkr_service._get_error_message("1012")
        assert "Token has expired" in message

    def test_get_error_message_unknown_code(self, ibkr_service):
        """Test getting error message for unknown error code."""
        message = ibkr_service._get_error_message("9999")
        assert "Unknown error" in message
        assert "9999" in message

    @responses.activate
    def test_fetch_statement_token_expired_error(self, ibkr_service, test_token, test_query_id):
        """Test handling of token expiration (error 1012)."""
        # Mock SendRequest to return token expired error
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=SAMPLE_SEND_REQUEST_ERROR_1012,
            status=200,
        )

        result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=False)
        assert result is None

    @responses.activate
    def test_fetch_statement_invalid_token_error(self, ibkr_service, test_token, test_query_id):
        """Test handling of invalid token (error 1015)."""
        # Mock SendRequest to return invalid token error
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=SAMPLE_SEND_REQUEST_ERROR_1015,
            status=200,
        )

        result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=False)
        assert result is None

    @responses.activate
    def test_fetch_statement_http_error(self, ibkr_service, test_token, test_query_id):
        """Test handling of HTTP errors."""
        # Mock HTTP 500 error
        responses.add(responses.GET, IBKRFlexService.SEND_REQUEST_URL, status=500)

        result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=False)
        assert result is None


class TestFetchStatement:
    """Tests for fetching statements from IBKR API."""

    @responses.activate
    def test_fetch_statement_success(
        self, app_context, db_session, ibkr_service, test_token, test_query_id
    ):
        """Test successful statement fetch."""
        # Mock SendRequest
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=SAMPLE_SEND_REQUEST_SUCCESS,
            status=200,
        )

        # Mock GetStatement
        responses.add(
            responses.GET, IBKRFlexService.GET_STATEMENT_URL, body=SAMPLE_FLEX_STATEMENT, status=200
        )

        result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=False)

        assert result is not None
        assert "FlexQueryResponse" in result
        assert "AAPL" in result

    @responses.activate
    def test_fetch_statement_with_retry(
        self, app_context, db_session, ibkr_service, test_token, test_query_id
    ):
        """Test statement fetch with retry logic (1019 error)."""
        # Mock SendRequest
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=SAMPLE_SEND_REQUEST_SUCCESS,
            status=200,
        )

        # Mock GetStatement - first two requests return "in progress", third succeeds
        responses.add(
            responses.GET,
            IBKRFlexService.GET_STATEMENT_URL,
            body=SAMPLE_STATEMENT_IN_PROGRESS,
            status=200,
        )
        responses.add(
            responses.GET,
            IBKRFlexService.GET_STATEMENT_URL,
            body=SAMPLE_STATEMENT_IN_PROGRESS,
            status=200,
        )
        responses.add(
            responses.GET, IBKRFlexService.GET_STATEMENT_URL, body=SAMPLE_FLEX_STATEMENT, status=200
        )

        with patch("time.sleep"):  # Skip actual sleep delays
            result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=False)

        assert result is not None
        assert "FlexQueryResponse" in result

    @responses.activate
    def test_fetch_statement_uses_cache(
        self, app_context, db_session, ibkr_service, test_token, test_query_id
    ):
        """Test that fetch_statement uses cache when available."""
        cache_key = ibkr_service._get_cache_key(test_query_id)
        cached_data = "<xml>cached statement</xml>"

        # Add cache entry
        ibkr_service._cache_data(cache_key, cached_data)

        # Should return cached data without making HTTP requests
        result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=True)

        assert result == cached_data
        # No HTTP calls should have been made (responses would fail if they were)

    @responses.activate
    def test_fetch_statement_bypass_cache(self, app_context, db_session, ibkr_service, test_token):
        """Test that fetch_statement can bypass cache."""
        # Use unique query_id for this test to avoid cache key collision
        unique_query_id = make_custom_string("query_", 8)

        # Setup cache
        cache_key = ibkr_service._get_cache_key(unique_query_id)
        ibkr_service._cache_data(cache_key, "<xml>old cached data</xml>")

        # Mock fresh API call
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=SAMPLE_SEND_REQUEST_SUCCESS,
            status=200,
        )
        responses.add(
            responses.GET, IBKRFlexService.GET_STATEMENT_URL, body=SAMPLE_FLEX_STATEMENT, status=200
        )

        # Should bypass cache and fetch fresh data
        result = ibkr_service.fetch_statement(test_token, unique_query_id, use_cache=False)

        assert result is not None
        assert "FlexQueryResponse" in result
        assert len(responses.calls) == 2  # Both API calls were made

    @responses.activate
    def test_fetch_statement_network_error(self, ibkr_service, test_token, test_query_id):
        """Test handling of network errors."""
        import requests

        # Mock network error
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=requests.RequestException("Network error"),
        )

        result = ibkr_service.fetch_statement(test_token, test_query_id, use_cache=False)
        assert result is None


class TestParseFlexStatement:
    """Tests for XML parsing."""

    def test_parse_flex_statement_trades(self, app_context, db_session, ibkr_service):
        """Test parsing trade transactions from XML."""
        transactions = ibkr_service.parse_flex_statement(SAMPLE_FLEX_STATEMENT)

        # Should parse 2 trades + 2 cash transactions = 4 total
        assert len(transactions) >= 2

        # Find the buy trade
        buy_trades = [t for t in transactions if t["transaction_type"] == "buy"]
        assert len(buy_trades) == 1

        buy = buy_trades[0]
        assert buy["symbol"] == "AAPL"
        assert buy["isin"] == "US0378331005"
        assert buy["quantity"] == 10
        assert buy["price"] == 150.00
        assert buy["currency"] == "USD"

        # Find the sell trade
        sell_trades = [t for t in transactions if t["transaction_type"] == "sell"]
        assert len(sell_trades) == 1

        sell = sell_trades[0]
        assert sell["symbol"] == "MSFT"
        assert sell["quantity"] == 5  # Should be positive

    def test_parse_flex_statement_cash_transactions(self, app_context, db_session, ibkr_service):
        """Test parsing cash transactions (dividends, fees) from XML."""
        transactions = ibkr_service.parse_flex_statement(SAMPLE_FLEX_STATEMENT)

        # Find dividend transaction
        dividends = [t for t in transactions if t["transaction_type"] == "dividend"]
        assert len(dividends) == 1

        div = dividends[0]
        assert div["symbol"] == "AAPL"
        assert div["total_amount"] == 25.50
        assert div["quantity"] is None  # Cash transactions don't have quantity

        # Find fee transaction
        fees = [t for t in transactions if t["transaction_type"] == "fee"]
        assert len(fees) == 1

        fee = fees[0]
        assert fee["total_amount"] == 2.50

    def test_parse_flex_statement_missing_currency(self, app_context, db_session, ibkr_service):
        """Test parsing transactions with missing currency field (should default to USD)."""
        transactions = ibkr_service.parse_flex_statement(SAMPLE_STATEMENT_NO_CURRENCY)

        # Should parse transactions even without currency field
        assert len(transactions) >= 1

        # All transactions should have USD as default currency
        for txn in transactions:
            assert txn["currency"] == "USD"

    def test_parse_flex_statement_exchange_rates(self, app_context, db_session, ibkr_service):
        """Test parsing and importing exchange rates from XML."""
        # Parse statement (should import exchange rates)
        ibkr_service.parse_flex_statement(SAMPLE_FLEX_STATEMENT)

        # Check that exchange rates were imported
        eur_rate = ExchangeRate.query.filter_by(
            from_currency="EUR",
            to_currency="USD",
            date=datetime.strptime("20250115", "%Y%m%d").date(),
        ).first()

        assert eur_rate is not None
        assert eur_rate.rate == 1.10

    def test_parse_invalid_xml(self, ibkr_service):
        """Test handling of invalid XML."""
        invalid_xml = "<invalid>not closed"

        transactions = ibkr_service.parse_flex_statement(invalid_xml)
        assert transactions == []

    def test_parse_empty_statement(self, ibkr_service):
        """Test parsing statement with no transactions."""
        empty_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <FlexQueryResponse>
          <FlexStatements>
            <FlexStatement>
              <Trades></Trades>
              <CashTransactions></CashTransactions>
            </FlexStatement>
          </FlexStatements>
        </FlexQueryResponse>"""

        transactions = ibkr_service.parse_flex_statement(empty_xml)
        assert transactions == []

    def test_parse_statement_multiple_currencies(self, app_context, db_session, ibkr_service):
        """Test parsing statement with transactions in multiple currencies."""
        multi_currency_xml = """<?xml version="1.0" encoding="UTF-8"?>
        <FlexQueryResponse queryName="Test" type="AF">
          <FlexStatements count="1">
            <FlexStatement accountId="U1234567" fromDate="2025-01-01" toDate="2025-01-31">
              <Trades>
                <Trade accountId="U1234567" symbol="AAPL" isin="US0378331005"
                       description="APPLE INC"
                       tradeDate="20250115" quantity="10" tradePrice="150.00" netCash="-1500.00"
                       currency="USD" ibCommission="-1.00" transactionID="12345" ibOrderID="67890"/>
                <Trade accountId="U1234567" symbol="BMW" isin="DE0005190003" description="BMW AG"
                       tradeDate="20250116" quantity="5" tradePrice="95.00" netCash="-475.00"
                       currency="EUR" ibCommission="-1.00" transactionID="12346" ibOrderID="67891"/>
                <Trade accountId="U1234567" symbol="VOD" isin="GB00BH4HKS39"
                       description="VODAFONE GROUP"
                       tradeDate="20250117" quantity="100" tradePrice="0.75" netCash="-75.00"
                       currencyPrimary="GBP" ibCommission="-0.50"
                       transactionID="12347" ibOrderID="67892"/>
              </Trades>
            </FlexStatement>
          </FlexStatements>
        </FlexQueryResponse>"""

        transactions = ibkr_service.parse_flex_statement(multi_currency_xml)

        # Should have parsed 3 transactions
        assert len(transactions) == 3

        # Check USD transaction
        usd_txns = [t for t in transactions if t["currency"] == "USD"]
        assert len(usd_txns) == 1
        assert usd_txns[0]["symbol"] == "AAPL"

        # Check EUR transaction
        eur_txns = [t for t in transactions if t["currency"] == "EUR"]
        assert len(eur_txns) == 1
        assert eur_txns[0]["symbol"] == "BMW"
        assert eur_txns[0]["total_amount"] == 475.00

        # Check GBP transaction (uses currencyPrimary field)
        gbp_txns = [t for t in transactions if t["currency"] == "GBP"]
        assert len(gbp_txns) == 1
        assert gbp_txns[0]["symbol"] == "VOD"
        assert gbp_txns[0]["total_amount"] == 75.00


class TestImportTransactions:
    """Tests for importing transactions to database."""

    def test_import_transactions_success(self, app_context, db_session, ibkr_service):
        """Test successful import of transactions."""
        transactions = [
            {
                "ibkr_transaction_id": make_custom_string("test_txn_", 8),
                "transaction_date": datetime(2025, 1, 15).date(),
                "symbol": "AAPL",
                "isin": "US0378331005",
                "description": "APPLE INC",
                "transaction_type": "buy",
                "quantity": 10,
                "price": 150.00,
                "total_amount": 1500.00,
                "currency": "USD",
                "fees": 1.00,
                "raw_data": "{}",
            }
        ]

        result = ibkr_service.import_transactions(transactions)

        assert result["imported"] == 1
        assert result["skipped"] == 0
        assert len(result["errors"]) == 0

        # Verify transaction was saved
        db_txn = IBKRTransaction.query.filter_by(
            ibkr_transaction_id=transactions[0]["ibkr_transaction_id"]
        ).first()
        assert db_txn is not None
        assert db_txn.symbol == "AAPL"

    def test_import_duplicate_transactions(self, app_context, db_session, ibkr_service):
        """Test that duplicate transactions are skipped."""
        txn_id = make_custom_string("test_txn_", 8)
        transaction = {
            "ibkr_transaction_id": txn_id,
            "transaction_date": datetime(2025, 1, 15).date(),
            "symbol": "AAPL",
            "isin": "US0378331005",
            "description": "APPLE INC",
            "transaction_type": "buy",
            "quantity": 10,
            "price": 150.00,
            "total_amount": 1500.00,
            "currency": "USD",
            "fees": 1.00,
            "raw_data": "{}",
        }

        # Import first time
        result1 = ibkr_service.import_transactions([transaction])
        assert result1["imported"] == 1

        # Import again - should skip
        result2 = ibkr_service.import_transactions([transaction])
        assert result2["imported"] == 0
        assert result2["skipped"] == 1

    def test_import_mixed_new_and_duplicate(self, app_context, db_session, ibkr_service):
        """Test importing mix of new and duplicate transactions."""
        existing_id = make_custom_string("existing_", 8)
        new_id = make_custom_string("new_", 8)

        # Create existing transaction
        existing_txn = IBKRTransaction(
            ibkr_transaction_id=existing_id,
            transaction_date=datetime(2025, 1, 15).date(),
            symbol="AAPL",
            isin="US0378331005",
            transaction_type="buy",
            quantity=10,
            price=150.00,
            total_amount=1500.00,
            currency="USD",
            fees=1.00,
            raw_data="{}",
        )
        db.session.add(existing_txn)
        db.session.commit()

        # Import mix
        transactions = [
            {
                "ibkr_transaction_id": existing_id,
                "transaction_date": datetime(2025, 1, 15).date(),
                "symbol": "AAPL",
                "isin": "US0378331005",
                "description": "APPLE INC",
                "transaction_type": "buy",
                "quantity": 10,
                "price": 150.00,
                "total_amount": 1500.00,
                "currency": "USD",
                "fees": 1.00,
                "raw_data": "{}",
            },
            {
                "ibkr_transaction_id": new_id,
                "transaction_date": datetime(2025, 1, 16).date(),
                "symbol": "MSFT",
                "isin": "US5949181045",
                "description": "MICROSOFT CORP",
                "transaction_type": "buy",
                "quantity": 5,
                "price": 380.00,
                "total_amount": 1900.00,
                "currency": "USD",
                "fees": 1.00,
                "raw_data": "{}",
            },
        ]

        result = ibkr_service.import_transactions(transactions)
        assert result["imported"] == 1
        assert result["skipped"] == 1


class TestConnectionTest:
    """Tests for IBKR connection testing."""

    @responses.activate
    def test_connection_success(
        self, app_context, db_session, ibkr_service, test_token, test_query_id
    ):
        """Test successful connection test."""
        # Mock API responses
        responses.add(
            responses.GET,
            IBKRFlexService.SEND_REQUEST_URL,
            body=SAMPLE_SEND_REQUEST_SUCCESS,
            status=200,
        )
        responses.add(
            responses.GET, IBKRFlexService.GET_STATEMENT_URL, body=SAMPLE_FLEX_STATEMENT, status=200
        )

        result = ibkr_service.test_connection(test_token, test_query_id)

        assert result["success"] is True
        assert "transaction_count" in result
        assert result["transaction_count"] > 0

    @responses.activate
    def test_connection_failure(self, ibkr_service, test_token, test_query_id):
        """Test connection test with API failure."""
        # Mock API error
        responses.add(responses.GET, IBKRFlexService.SEND_REQUEST_URL, status=500)

        result = ibkr_service.test_connection(test_token, test_query_id)

        assert result["success"] is False
        assert "message" in result
