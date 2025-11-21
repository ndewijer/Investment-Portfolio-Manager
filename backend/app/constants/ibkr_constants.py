"""
IBKR Flex Web Service Constants.

This module contains constants for the IBKR Flex Web Service API,
including endpoints, error codes, and configuration values.

References:
- IBKR Flex Web Service: https://www.interactivebrokers.com/campus/ibkr-api-page/flex-web-service/
- Error Codes: https://www.interactivebrokers.com/campus/ibkr-api-page/flex-web-service/#error-codes
"""

# IBKR Flex API endpoints (per official documentation)
# https://www.interactivebrokers.com/campus/ibkr-api-page/flex-web-service/
FLEX_SEND_REQUEST_URL = (
    "https://ndcdyn.interactivebrokers.com/AccountManagement/FlexWebService/SendRequest"
)
FLEX_GET_STATEMENT_URL = (
    "https://ndcdyn.interactivebrokers.com/AccountManagement/FlexWebService/GetStatement"
)

# Cache settings
FLEX_CACHE_DURATION_MINUTES = 15

# IBKR Flex API Error Codes
# Source: https://www.interactivebrokers.com/campus/ibkr-api-page/flex-web-service/#error-codes
FLEX_ERROR_CODES = {
    "1001": "Statement could not be generated at this time. Please try again shortly.",
    "1003": "Statement is not available.",
    "1004": "Statement is incomplete at this time. Please try again shortly.",
    "1005": "Settlement data is not ready at this time. Please try again shortly.",
    "1006": "FIFO P/L data is not ready at this time. Please try again shortly.",
    "1007": "MTM P/L data is not ready at this time. Please try again shortly.",
    "1008": "MTM and FIFO P/L data is not ready at this time. Please try again shortly.",
    "1009": (
        "The server is under heavy load. Statement could not be generated "
        "at this time. Please try again shortly."
    ),
    "1010": "Legacy Flex Queries are no longer supported. Please convert over to Activity Flex.",
    "1011": "Service account is inactive.",
    "1012": "Token has expired.",
    "1013": "IP restriction.",
    "1014": "Query is invalid.",
    "1015": "Token is invalid.",
    "1016": "Account is invalid.",
    "1017": "Reference code is invalid.",
    "1018": (
        "Too many requests have been made from this token. Please try again shortly. "
        "(Limited to one request per second, 10 requests per minute per token)"
    ),
    "1019": "Statement generation in progress. Please try again shortly.",
    "1020": "Invalid request or unable to validate request.",
    "1021": "Statement could not be retrieved at this time. Please try again shortly.",
}
