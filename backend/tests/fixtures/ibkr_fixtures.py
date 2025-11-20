"""
Shared IBKR API fixtures for testing.

These fixtures are used by both service and route tests to mock IBKR Flex API responses.
Centralizing them here ensures consistency and reduces duplication.

Usage:
    from tests.fixtures.ibkr_fixtures import SAMPLE_FLEX_STATEMENT, SAMPLE_SEND_REQUEST_SUCCESS
"""

# Sample XML fixtures for IBKR Flex API responses

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
