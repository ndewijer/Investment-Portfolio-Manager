"""
Service for IBKR Flex Web Service integration.

This module handles:
- Fetching transaction data from IBKR Flex API
- Parsing XML responses
- Caching API responses to avoid duplicate requests
- Transforming IBKR data to internal format
"""

import contextlib
import json
import os
import sys
import xml.etree.ElementTree as ET
from datetime import datetime, timedelta

import requests
from cryptography.fernet import Fernet

from ..constants.ibkr_constants import (
    FLEX_CACHE_DURATION_MINUTES,
    FLEX_ERROR_CODES,
    FLEX_GET_STATEMENT_URL,
    FLEX_SEND_REQUEST_URL,
)
from ..models import IBKRImportCache, IBKRTransaction, LogCategory, LogLevel, db
from ..services.logging_service import logger


class IBKRFlexService:
    """
    Handle IBKR Flex Web Service API calls and data processing.

    This service manages the complete workflow of:
    1. Requesting statements from IBKR
    2. Polling for completion
    3. Parsing XML responses
    4. Caching results
    5. Transforming to internal format
    """

    def __init__(self):
        """Initialize IBKR Flex Service."""
        from flask import current_app

        # Get encryption key from app config (auto-generated if not in env)
        self.encryption_key = current_app.config.get("IBKR_ENCRYPTION_KEY")

        # Debug XML saving (disabled by default for security)
        self.save_debug_xml = os.environ.get("IBKR_DEBUG_SAVE_XML", "false").lower() == "true"

        # Set User-Agent header (required by IBKR Flex Web Service for programmatic access)
        python_version = (
            f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"
        )
        self.headers = {"User-Agent": f"Python/{python_version}"}

        logger.log(
            level=LogLevel.DEBUG,
            category=LogCategory.IBKR,
            message="IBKRFlexService initialized",
            details={
                "python_version": python_version,
                "headers": self.headers,
                "save_debug_xml": self.save_debug_xml,
            },
        )

        if not self.encryption_key:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SECURITY,
                message="IBKR encryption key not available",
                details={
                    "error": "Encryption key not found in app config",
                    "solution": "Check application startup logs for key generation",
                },
            )

    def _get_error_message(self, error_code: str) -> str:
        """
        Get verbose error message for IBKR error code.

        Args:
            error_code: IBKR error code (e.g., "1019")

        Returns:
            Verbose error message, or generic message if code unknown
        """
        return FLEX_ERROR_CODES.get(
            error_code, f"Unknown error (code: {error_code}). Please check IBKR documentation."
        )

    def _map_flex_error_to_http_status(self, error_code: str) -> int:
        """
        Map IBKR Flex error codes to appropriate HTTP status codes.

        Args:
            error_code: IBKR error code (e.g., "1009")

        Returns:
            HTTP status code (400, 401, 403, 429, or 500)
        """
        # Server errors - IBKR infrastructure issues
        server_errors = [
            "1001",
            "1003",
            "1004",
            "1005",
            "1006",
            "1007",
            "1008",
            "1009",
            "1019",
            "1021",
        ]

        # Forbidden - Access/permission issues
        forbidden_errors = ["1011", "1013", "1014", "1015", "1016"]

        # Unauthorized - Token expired
        unauthorized_errors = ["1012"]

        # Rate limiting - Too many requests
        rate_limit_errors = ["1018"]

        if error_code in server_errors:
            return 500
        elif error_code in forbidden_errors:
            return 403
        elif error_code in unauthorized_errors:
            return 401
        elif error_code in rate_limit_errors:
            return 429
        else:
            # Default for bad requests (like 1010, 1017, 1020)
            return 400

    def _encrypt_token(self, token: str) -> str:
        """
        Encrypt IBKR token.

        Args:
            token: Plain text token

        Returns:
            Encrypted token string
        """
        if not self.encryption_key:
            raise ValueError("Encryption key not available")

        # Strip whitespace from token before encrypting
        token = token.strip()
        f = Fernet(self.encryption_key.encode())
        return f.encrypt(token.encode()).decode()

    def _decrypt_token(self, encrypted_token: str) -> str:
        """
        Decrypt IBKR token.

        Args:
            encrypted_token: Encrypted token

        Returns:
            Plain text token string
        """
        if not self.encryption_key:
            raise ValueError("Encryption key not available")

        f = Fernet(self.encryption_key.encode())
        # Strip whitespace from decrypted token
        return f.decrypt(encrypted_token.encode()).decode().strip()

    def _get_cache_key(self, query_id: str) -> str:
        """
        Generate cache key for query.

        Args:
            query_id: IBKR query ID

        Returns:
            Cache key string
        """
        # Include current date to ensure we get latest data daily
        today = datetime.now().strftime("%Y-%m-%d")
        return f"ibkr_flex_{query_id}_{today}"

    def _get_cached_data(self, cache_key: str) -> str | None:
        """
        Retrieve cached data if available and not expired.

        Args:
            cache_key: Cache key

        Returns:
            Cached data or None if not found/expired
        """
        cache_entry = IBKRImportCache.query.filter_by(cache_key=cache_key).first()

        if cache_entry and cache_entry.expires_at > datetime.now():
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Using cached IBKR data for key: {cache_key}",
                details={"cache_key": cache_key, "expires_at": cache_entry.expires_at.isoformat()},
            )
            # Save cached data to debug folder too for inspection
            self._save_debug_xml(cache_entry.data, f"cached_{cache_key}")
            return cache_entry.data

        # Clean up expired cache entry if exists
        if cache_entry:
            db.session.delete(cache_entry)
            db.session.commit()

        return None

    def _cache_data(self, cache_key: str, data: str) -> None:
        """
        Store data in cache.

        Args:
            cache_key: Cache key
            data: Data to cache
        """
        expires_at = datetime.now() + timedelta(minutes=FLEX_CACHE_DURATION_MINUTES)

        cache_entry = IBKRImportCache(cache_key=cache_key, data=data, expires_at=expires_at)
        db.session.add(cache_entry)
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Cached IBKR data for key: {cache_key}",
            details={"cache_key": cache_key, "expires_at": expires_at.isoformat()},
        )

    def _clean_expired_cache(self) -> None:
        """Delete expired cache entries."""
        expired_entries = IBKRImportCache.query.filter(
            IBKRImportCache.expires_at < datetime.now()
        ).all()

        if expired_entries:
            for entry in expired_entries:
                db.session.delete(entry)
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Cleaned {len(expired_entries)} expired cache entries",
                details={"count": len(expired_entries)},
            )

    def _save_debug_xml(self, xml_content: str, reference_code: str) -> None:
        """
        Save XML to disk for debugging purposes.

        Controlled by IBKR_DEBUG_SAVE_XML environment variable (disabled by default).

        Args:
            xml_content: XML content to save
            reference_code: IBKR reference code for the statement
        """
        # Only save if debug mode is explicitly enabled
        if not self.save_debug_xml:
            return

        import os

        try:
            # Create debug directory if it doesn't exist
            debug_dir = os.path.join(os.path.dirname(__file__), "../../data/ibkr_debug")
            os.makedirs(debug_dir, exist_ok=True)

            # Generate filename with timestamp
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"ibkr_statement_{timestamp}_{reference_code}.xml"
            filepath = os.path.join(debug_dir, filename)

            # Save XML to file
            with open(filepath, "w", encoding="utf-8") as f:
                f.write(xml_content)

            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message=f"Saved debug XML to {filepath}",
                details={"filepath": filepath, "reference_code": reference_code},
            )
        except Exception as e:
            logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.IBKR,
                message="Failed to save debug XML",
                details={"error": str(e)},
            )

    def fetch_statement(self, token: str, query_id: str, use_cache: bool = True) -> str | None:
        """
        Fetch Flex statement from IBKR.

        Args:
            token: IBKR Flex token
            query_id: IBKR Flex query ID
            use_cache: Whether to use cached data if available

        Returns:
            XML statement data or None on error
        """
        cache_key = self._get_cache_key(query_id)

        # Check cache first if enabled
        if use_cache:
            cached_data = self._get_cached_data(cache_key)
            if cached_data:
                return cached_data

        try:
            # Step 1: Request the statement
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="Requesting Flex statement from IBKR",
                details={
                    "query_id": query_id,
                    "url": FLEX_SEND_REQUEST_URL,
                    "headers": self.headers,
                    "params": {"q": query_id, "v": "3", "t": "***"},
                },
            )

            response = requests.get(
                FLEX_SEND_REQUEST_URL,
                params={"t": token, "q": query_id, "v": "3"},
                headers=self.headers,
                timeout=30,
            )

            # Debug: Log response details
            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message="IBKR SendRequest response received",
                details={
                    "status_code": response.status_code,
                    "response_headers": dict(response.headers),
                    "request_headers": dict(response.request.headers),
                },
            )

            if response.status_code != 200:
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message="Failed to request IBKR statement",
                    details={"status_code": response.status_code, "response": response.text},
                )
                return None

            # Parse response to get reference code
            root = ET.fromstring(response.text)
            status = root.find("Status").text if root.find("Status") is not None else None
            reference_code = (
                root.find("ReferenceCode").text if root.find("ReferenceCode") is not None else None
            )
            error_code = root.find("ErrorCode").text if root.find("ErrorCode") is not None else None

            if status != "Success" or not reference_code:
                error_msg = self._get_error_message(error_code) if error_code else "Unknown error"
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message=f"IBKR request failed: {error_msg}",
                    details={
                        "status": status,
                        "error_code": error_code,
                        "response": response.text,
                    },
                )
                return None

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Received reference code: {reference_code}",
                details={"reference_code": reference_code},
            )

            # Step 2: Poll for statement with retry logic
            max_retries = 10
            retry_delay = 2  # seconds
            import time

            for attempt in range(max_retries):
                statement_response = requests.get(
                    FLEX_GET_STATEMENT_URL,
                    params={"q": reference_code, "t": token, "v": "3"},
                    headers=self.headers,
                    timeout=30,
                )

                if statement_response.status_code != 200:
                    logger.log(
                        level=LogLevel.ERROR,
                        category=LogCategory.IBKR,
                        message="Failed to retrieve IBKR statement",
                        details={
                            "status_code": statement_response.status_code,
                            "response": statement_response.text,
                        },
                    )
                    return None

                # Check response format - could be XML or JSON
                response_text = statement_response.text.strip()

                # If response starts with '<', it's XML
                if response_text.startswith("<"):
                    statement_root = ET.fromstring(response_text)
                    # Handle both response types: FlexStatementResponse (correct endpoint)
                    # or FlexQueryResponse (legacy endpoint)
                    if statement_root.tag in ["FlexStatementResponse", "FlexQueryResponse"]:
                        # Check status
                        status_elem = statement_root.find("Status")
                        error_code_elem = statement_root.find("ErrorCode")

                        # Check if statement is still generating (error code 1019)
                        if (
                            status_elem is not None
                            and status_elem.text == "Warn"
                            and error_code_elem is not None
                            and error_code_elem.text == "1019"
                        ):
                            # Statement generation in progress - retry
                            if attempt < max_retries - 1:
                                logger.log(
                                    level=LogLevel.INFO,
                                    category=LogCategory.IBKR,
                                    message=(
                                        f"Statement generation in progress, "
                                        f"retrying in {retry_delay}s "
                                        f"(attempt {attempt + 1}/{max_retries})"
                                    ),
                                    details={
                                        "reference_code": reference_code,
                                        "attempt": attempt + 1,
                                    },
                                )
                                time.sleep(retry_delay)
                                continue
                            else:
                                logger.log(
                                    level=LogLevel.ERROR,
                                    category=LogCategory.IBKR,
                                    message="Statement generation timeout after max retries",
                                    details={
                                        "reference_code": reference_code,
                                        "max_retries": max_retries,
                                    },
                                )
                                return None

                        # Check for other errors
                        if status_elem is not None and status_elem.text in ["Warn", "Fail"]:
                            error_code = (
                                error_code_elem.text if error_code_elem is not None else None
                            )
                            # Get verbose error message from our mapping
                            error_msg = self._get_error_message(error_code) if error_code else None
                            # Fallback to IBKR's error message if provided
                            if not error_msg:
                                error_message = statement_root.find("ErrorMessage")
                                error_msg = (
                                    error_message.text
                                    if error_message is not None
                                    else "Unknown error"
                                )
                            logger.log(
                                level=LogLevel.ERROR,
                                category=LogCategory.IBKR,
                                message=f"IBKR returned error status: {error_msg}",
                                details={
                                    "status": status_elem.text,
                                    "error_code": error_code,
                                },
                            )
                            return None

                        # Statement is ready and successful
                        logger.log(
                            level=LogLevel.INFO,
                            category=LogCategory.IBKR,
                            message="Successfully retrieved Flex statement",
                            details={"reference_code": reference_code, "attempts": attempt + 1},
                        )

                        # Save XML to disk for debugging
                        self._save_debug_xml(response_text, reference_code)

                        # Cache the response (only cache successful responses)
                        if use_cache:
                            self._cache_data(cache_key, response_text)

                        return response_text

                # If we get here without returning, statement wasn't in expected format
                # Continue to next retry
                if attempt < max_retries - 1:
                    logger.log(
                        level=LogLevel.WARNING,
                        category=LogCategory.IBKR,
                        message="Unexpected response format, retrying",
                        details={"attempt": attempt + 1, "response_preview": response_text[:200]},
                    )
                    time.sleep(retry_delay)
                    continue

            # If we exhausted all retries without success
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to retrieve statement after all retries",
                details={"reference_code": reference_code, "max_retries": max_retries},
            )
            return None

        except requests.RequestException as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Network error fetching IBKR statement",
                details={"error": str(e)},
            )
            return None
        except ET.ParseError as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to parse IBKR XML response",
                details={"error": str(e)},
            )
            return None
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Unexpected error fetching IBKR statement",
                details={"error": str(e)},
            )
            return None

    def parse_flex_statement(self, xml_data: str) -> list[dict]:
        """
        Parse IBKR Flex statement XML into transaction records.

        Args:
            xml_data: XML statement data from IBKR

        Returns:
            List of transaction dictionaries
        """
        transactions = []

        try:
            root = ET.fromstring(xml_data)

            # Parse trades
            trades = root.findall(".//Trade")
            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message=f"Found {len(trades)} trades in XML",
                details={"trade_count": len(trades)},
            )

            for trade in trades:
                transaction = self._parse_trade(trade)
                if transaction:
                    transactions.append(transaction)
                    logger.log(
                        level=LogLevel.DEBUG,
                        category=LogCategory.IBKR,
                        message="Parsed trade successfully",
                        details={
                            "symbol": transaction.get("symbol"),
                            "ibkr_transaction_id": transaction.get("ibkr_transaction_id"),
                        },
                    )

            # Parse cash transactions (dividends, fees)
            cash_transactions = root.findall(".//CashTransaction")
            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message=f"Found {len(cash_transactions)} cash transactions in XML",
                details={"cash_transaction_count": len(cash_transactions)},
            )
            for cash_txn in cash_transactions:
                transaction = self._parse_cash_transaction(cash_txn)
                if transaction:
                    transactions.append(transaction)

            # Parse and import exchange rates
            self._import_exchange_rates(root)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Parsed {len(transactions)} transactions from IBKR statement",
                details={"count": len(transactions)},
            )

            return transactions

        except ET.ParseError as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to parse IBKR XML",
                details={"error": str(e)},
            )
            return []
        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Unexpected error parsing IBKR statement",
                details={"error": str(e)},
            )
            return []

    def _import_exchange_rates(self, root: ET.Element) -> None:
        """
        Parse and import exchange rates from IBKR XML.

        Args:
            root: Root XML element
        """
        from datetime import datetime

        from ..models import ExchangeRate, db

        try:
            conversion_rates = root.findall(".//ConversionRate")
            imported_count = 0
            skipped_count = 0

            for rate_element in conversion_rates:
                from_currency = rate_element.get("fromCurrency")
                to_currency = rate_element.get("toCurrency")
                rate = float(rate_element.get("rate", 0))
                date_str = rate_element.get("reportDate")

                if not all([from_currency, to_currency, rate, date_str]):
                    continue

                # Parse date (format: YYYYMMDD)
                try:
                    date = datetime.strptime(date_str, "%Y%m%d").date()
                except ValueError:
                    continue

                # Check if rate already exists
                existing_rate = ExchangeRate.query.filter_by(
                    from_currency=from_currency, to_currency=to_currency, date=date
                ).first()

                if existing_rate:
                    skipped_count += 1
                    continue

                # Create new exchange rate
                new_rate = ExchangeRate(
                    from_currency=from_currency, to_currency=to_currency, rate=rate, date=date
                )
                db.session.add(new_rate)
                imported_count += 1

            if imported_count > 0:
                db.session.commit()
                logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.IBKR,
                    message=(
                        f"Imported {imported_count} exchange rates, "
                        f"skipped {skipped_count} duplicates"
                    ),
                    details={"imported": imported_count, "skipped": skipped_count},
                )

        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to import exchange rates",
                details={"error": str(e)},
            )

    def _parse_trade(self, trade_element: ET.Element) -> dict | None:
        """
        Parse a trade element from IBKR XML.

        Args:
            trade_element: XML element for trade

        Returns:
            Transaction dictionary or None if invalid
        """
        try:
            # Extract trade data
            symbol = trade_element.get("symbol")
            isin = trade_element.get("isin")
            description = trade_element.get("description", "")
            date_str = trade_element.get("tradeDate")
            report_date_str = trade_element.get("reportDate")
            quantity = float(trade_element.get("quantity", 0))
            price = float(trade_element.get("tradePrice", 0))
            amount = float(trade_element.get("netCash", 0))  # Negative for purchases
            # IBKR may use either 'currency' or 'currencyPrimary'
            currency = trade_element.get("currencyPrimary") or trade_element.get("currency", "USD")
            commission = abs(float(trade_element.get("ibCommission", 0)))
            buy_sell = trade_element.get("buySell")  # Used to determine type; not stored
            notes = trade_element.get("notes") or trade_element.get("Notes") or ""

            # Generate unique transaction ID
            ibkr_transaction_id = (
                f"{trade_element.get('transactionID', '')}_{trade_element.get('ibOrderID', '')}"
            )

            # Determine transaction type: use buySell if available, else fall back to quantity sign
            if buy_sell:
                buy_sell_upper = buy_sell.upper()
                if buy_sell_upper in ("BUY", "B"):
                    transaction_type = "buy"
                    quantity = abs(quantity)
                elif buy_sell_upper in ("SELL", "S"):
                    transaction_type = "sell"
                    quantity = abs(quantity)
                else:
                    return None
            else:
                if quantity > 0:
                    transaction_type = "buy"
                elif quantity < 0:
                    transaction_type = "sell"
                    quantity = abs(quantity)
                else:
                    return None

            # Parse trade date
            transaction_date = datetime.strptime(date_str, "%Y%m%d").date()

            # Parse report date if available
            report_date = None
            if report_date_str:
                with contextlib.suppress(ValueError):
                    report_date = datetime.strptime(report_date_str, "%Y%m%d").date()

            return {
                "ibkr_transaction_id": ibkr_transaction_id,
                "transaction_date": transaction_date,
                "report_date": report_date or transaction_date,
                "symbol": symbol,
                "isin": isin,
                "description": description,
                "transaction_type": transaction_type,
                "quantity": quantity,
                "price": abs(price),
                "total_amount": abs(amount),
                "currency": currency,
                "fees": commission,
                "notes": notes,
                "raw_data": json.dumps(trade_element.attrib),
            }

        except (ValueError, AttributeError) as e:
            logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.IBKR,
                message="Failed to parse trade element",
                details={"error": str(e), "element": str(ET.tostring(trade_element))},
            )
            return None

    def _parse_cash_transaction(self, cash_element: ET.Element) -> dict | None:
        """
        Parse a cash transaction element from IBKR XML.

        Args:
            cash_element: XML element for cash transaction

        Returns:
            Transaction dictionary or None if invalid
        """
        try:
            transaction_type_raw = cash_element.get("type", "")
            symbol = cash_element.get("symbol")
            isin = cash_element.get("isin")
            description = cash_element.get("description", "")
            report_date_str = cash_element.get("reportDate")
            date_str = report_date_str or cash_element.get("dateTime", "")
            amount = float(cash_element.get("amount", 0))
            # IBKR may use either 'currency' or 'currencyPrimary'
            currency = cash_element.get("currencyPrimary") or cash_element.get("currency", "USD")
            code = cash_element.get("code", "")  # Transaction classification code
            ex_date = cash_element.get("exDate")  # Ex-dividend date

            # Generate unique transaction ID
            ibkr_transaction_id = cash_element.get("transactionID", "")
            if not ibkr_transaction_id:
                # Fallback to generating ID from attributes
                ibkr_transaction_id = f"cash_{date_str}_{symbol}_{amount}"

            # Determine transaction type
            if "Dividend" in transaction_type_raw or "DIV" in transaction_type_raw:
                transaction_type = "dividend"
            elif (
                "Commission" in transaction_type_raw
                or "Fee" in transaction_type_raw
                or "FEE" in transaction_type_raw
            ):
                transaction_type = "fee"
            else:
                # Skip other cash transaction types
                return None

            # Parse transaction date (handle both date formats)
            date_for_parsing = date_str
            if ";" in date_for_parsing:
                date_for_parsing = date_for_parsing.split(";")[0]
            if len(date_for_parsing) == 8:
                transaction_date = datetime.strptime(date_for_parsing, "%Y%m%d").date()
            else:
                transaction_date = datetime.strptime(
                    date_for_parsing.split(",")[0], "%Y-%m-%d"
                ).date()

            # Parse report date if available (separate from transaction date)
            report_date = None
            if report_date_str:
                with contextlib.suppress(ValueError):
                    report_date_clean = report_date_str.split(";")[0]
                    if len(report_date_clean) == 8:
                        report_date = datetime.strptime(report_date_clean, "%Y%m%d").date()

            # Enhance description with additional details
            enhanced_description = description
            if code:
                enhanced_description += f" [Code: {code}]"
            if ex_date and transaction_type == "dividend":
                enhanced_description += f" [Ex-Date: {ex_date}]"

            return {
                "ibkr_transaction_id": ibkr_transaction_id,
                "transaction_date": transaction_date,
                "report_date": report_date or transaction_date,
                "symbol": symbol,
                "isin": isin,
                "description": enhanced_description,
                "transaction_type": transaction_type,
                "quantity": None,  # No quantity for cash transactions
                "price": None,
                "total_amount": abs(amount),
                "currency": currency,
                "fees": 0.0,
                "notes": code or "",
                "raw_data": json.dumps(cash_element.attrib),
            }

        except (ValueError, AttributeError) as e:
            logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.IBKR,
                message="Failed to parse cash transaction element",
                details={"error": str(e), "element": str(ET.tostring(cash_element))},
            )
            return None

    def import_transactions(self, transactions: list[dict]) -> dict:
        """
        Import transactions into database, skipping duplicates.

        Args:
            transactions: List of transaction dictionaries

        Returns:
            Dictionary with import statistics
        """
        results = {"imported": 0, "skipped": 0, "errors": []}

        for txn_data in transactions:
            try:
                # Check for duplicate
                existing = IBKRTransaction.query.filter_by(
                    ibkr_transaction_id=txn_data["ibkr_transaction_id"]
                ).first()

                if existing:
                    results["skipped"] += 1
                    logger.log(
                        level=LogLevel.DEBUG,
                        category=LogCategory.IBKR,
                        message=(
                            f"Skipping duplicate transaction: {txn_data['ibkr_transaction_id']}"
                        ),
                        details={"ibkr_transaction_id": txn_data["ibkr_transaction_id"]},
                    )
                    continue

                # Create new transaction
                ibkr_txn = IBKRTransaction(**txn_data)
                db.session.add(ibkr_txn)
                results["imported"] += 1

            except Exception as e:
                results["errors"].append(
                    {
                        "ibkr_transaction_id": txn_data.get("ibkr_transaction_id"),
                        "error": str(e),
                    }
                )
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message="Error importing transaction",
                    details={
                        "ibkr_transaction_id": txn_data.get("ibkr_transaction_id"),
                        "error": str(e),
                    },
                )

        try:
            db.session.commit()
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Import complete: {results['imported']} imported, "
                f"{results['skipped']} skipped, {len(results['errors'])} errors",
                details=results,
            )
        except Exception as e:
            db.session.rollback()
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to commit imported transactions",
                details={"error": str(e)},
            )
            results["errors"].append({"error": "Database commit failed", "details": str(e)})

        # Clean up expired cache entries
        self._clean_expired_cache()

        return results

    def trigger_manual_import(self, config) -> tuple[dict, int]:
        """
        Trigger a manual IBKR transaction import.

        This method orchestrates the full import process:
        1. Decrypts the token
        2. Fetches the statement from IBKR
        3. Parses transactions
        4. Imports them to database
        5. Updates the last import date

        Args:
            config: IBKRConfig object with credentials

        Returns:
            tuple: (response dict, status code)
        """
        try:
            # Decrypt token
            token = self._decrypt_token(config.flex_token)

            # Fetch statement
            xml_data = self.fetch_statement(token, config.flex_query_id, use_cache=True)

            if not xml_data:
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message="Failed to fetch statement from IBKR",
                    details={"query_id": config.flex_query_id},
                )
                return {"error": "Failed to fetch statement from IBKR"}, 500

            # Parse transactions
            transactions = self.parse_flex_statement(xml_data)

            # Import transactions
            results = self.import_transactions(transactions)

            # Update last import date
            config.last_import_date = datetime.now()
            db.session.commit()

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="Manual IBKR import completed",
                details=results,
            )

            return (
                {
                    "success": True,
                    "message": "Import completed",
                    "imported": results["imported"],
                    "skipped": results["skipped"],
                    "errors": results["errors"],
                },
                200,
            )

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error during IBKR import",
                details={"error": str(e)},
            )
            return {"error": "Import failed", "details": str(e)}, 500

    def _test_connection_request(self, token: str, query_id: str) -> dict:
        """
        Perform a single connection test request and extract error information.

        Args:
            token: IBKR Flex token
            query_id: IBKR Flex query ID

        Returns:
            Dictionary with detailed test results including error codes
        """
        try:
            # Step 1: Test the initial request
            response = requests.get(
                FLEX_SEND_REQUEST_URL,
                params={"t": token, "q": query_id, "v": "3"},
                headers=self.headers,
                timeout=30,
            )

            if response.status_code != 200:
                return {
                    "success": False,
                    "message": "Failed to connect to IBKR API",
                    "http_status": 500,
                    "details": f"HTTP {response.status_code}: {response.text}",
                }

            # Parse response to check for errors
            root = ET.fromstring(response.text)
            status = root.find("Status").text if root.find("Status") is not None else None
            error_code = root.find("ErrorCode").text if root.find("ErrorCode") is not None else None

            if status != "Success":
                error_msg = self._get_error_message(error_code) if error_code else "Unknown error"
                http_status = self._map_flex_error_to_http_status(error_code) if error_code else 400

                return {
                    "success": False,
                    "message": error_msg,
                    "error_code": error_code,
                    "http_status": http_status,
                }

            # Success - we got a reference code
            reference_code = (
                root.find("ReferenceCode").text if root.find("ReferenceCode") is not None else None
            )

            return {
                "success": True,
                "message": "Connection successful",
                "reference_code": reference_code,
                "http_status": 200,
            }

        except requests.RequestException as e:
            return {
                "success": False,
                "message": "Network error connecting to IBKR",
                "http_status": 500,
                "details": str(e),
            }
        except ET.ParseError as e:
            return {
                "success": False,
                "message": "Invalid response from IBKR",
                "http_status": 500,
                "details": str(e),
            }
        except Exception as e:
            return {
                "success": False,
                "message": "Unexpected error during connection test",
                "http_status": 500,
                "details": str(e),
            }

    def test_connection(self, token: str, query_id: str) -> dict:
        """
        Test IBKR Flex connection.

        Args:
            token: IBKR Flex token
            query_id: IBKR Flex query ID

        Returns:
            Dictionary with test results including appropriate HTTP status code
        """
        try:
            # Strip whitespace from token for consistency
            token = token.strip()
            query_id = query_id.strip()

            # Use the detailed connection test
            result = self._test_connection_request(token, query_id)

            if not result["success"]:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.IBKR,
                    message=f"IBKR connection test failed: {result['message']}",
                    details={
                        "query_id": query_id,
                        "error_code": result.get("error_code"),
                        "http_status": result.get("http_status"),
                    },
                )
                return result

            # Connection successful - now fetch and parse the statement to verify full functionality
            xml_data = self.fetch_statement(token, query_id, use_cache=False)

            if not xml_data:
                return {
                    "success": False,
                    "message": "Connection failed: Unable to fetch statement",
                    "http_status": 500,
                }

            # Parse the XML to count transactions
            root = ET.fromstring(xml_data)
            trade_count = len(root.findall(".//Trade"))
            cash_count = len(root.findall(".//CashTransaction"))
            transaction_count = trade_count + cash_count

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="IBKR connection test successful",
                details={
                    "query_id": query_id,
                    "reference_code": result.get("reference_code"),
                    "transaction_count": transaction_count,
                },
            )

            return {
                "success": True,
                "message": "Connection successful",
                "reference_code": result.get("reference_code"),
                "transaction_count": transaction_count,
                "http_status": 200,
            }

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error testing IBKR connection",
                details={"error": str(e)},
            )
            return {
                "success": False,
                "message": f"Connection test failed: {e!s}",
                "http_status": 500,
            }
