"""
Service for IBKR Flex Web Service integration.

This module handles:
- Fetching transaction data from IBKR Flex API
- Parsing XML responses
- Caching API responses to avoid duplicate requests
- Transforming IBKR data to internal format
"""

import json
import os
import xml.etree.ElementTree as ET
from datetime import UTC, datetime, timedelta
from typing import Dict, List, Optional

import requests
from cryptography.fernet import Fernet

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

    # IBKR Flex API endpoints
    SEND_REQUEST_URL = (
        "https://gdcdyn.interactivebrokers.com/Universal/servlet/FlexStatementService.SendRequest"
    )
    GET_STATEMENT_URL = (
        "https://gdcdyn.interactivebrokers.com/Universal/servlet/FlexStatementService.GetStatement"
    )

    # Cache settings
    CACHE_DURATION_HOURS = 24

    def __init__(self):
        """Initialize IBKR Flex Service."""
        self.encryption_key = os.environ.get("IBKR_ENCRYPTION_KEY")
        if not self.encryption_key:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="IBKR_ENCRYPTION_KEY not set in environment",
                details={"error": "Missing required environment variable"},
            )

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
        return f.decrypt(encrypted_token.encode()).decode()

    def _get_cache_key(self, query_id: str) -> str:
        """
        Generate cache key for query.

        Args:
            query_id: IBKR query ID

        Returns:
            Cache key string
        """
        # Include current date to ensure we get latest data daily
        today = datetime.now(UTC).strftime("%Y-%m-%d")
        return f"ibkr_flex_{query_id}_{today}"

    def _get_cached_data(self, cache_key: str) -> Optional[str]:
        """
        Retrieve cached data if available and not expired.

        Args:
            cache_key: Cache key

        Returns:
            Cached data or None if not found/expired
        """
        cache_entry = IBKRImportCache.query.filter_by(cache_key=cache_key).first()

        if cache_entry and cache_entry.expires_at > datetime.now(UTC):
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message=f"Using cached IBKR data for key: {cache_key}",
                details={"cache_key": cache_key, "expires_at": cache_entry.expires_at.isoformat()},
            )
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
        expires_at = datetime.now(UTC) + timedelta(hours=self.CACHE_DURATION_HOURS)

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
            IBKRImportCache.expires_at < datetime.now(UTC)
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

    def fetch_statement(self, token: str, query_id: str, use_cache: bool = True) -> Optional[str]:
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
                details={"query_id": query_id},
            )

            response = requests.get(
                self.SEND_REQUEST_URL, params={"t": token, "q": query_id, "v": "3"}, timeout=30
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
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message="IBKR request failed",
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

            # Step 2: Poll for statement (typically ready immediately)
            statement_response = requests.get(
                self.GET_STATEMENT_URL,
                params={"q": reference_code, "t": token, "v": "3"},
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

            # Check if statement is ready
            statement_root = ET.fromstring(statement_response.text)
            if statement_root.tag == "FlexStatementResponse":
                # Statement is ready
                logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.IBKR,
                    message="Successfully retrieved Flex statement",
                    details={"reference_code": reference_code},
                )

                # Cache the response
                if use_cache:
                    self._cache_data(cache_key, statement_response.text)

                return statement_response.text
            else:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.IBKR,
                    message="Statement not ready yet",
                    details={"reference_code": reference_code, "response": statement_response.text},
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

    def parse_flex_statement(self, xml_data: str) -> List[Dict]:
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
            for trade in trades:
                transaction = self._parse_trade(trade)
                if transaction:
                    transactions.append(transaction)

            # Parse cash transactions (dividends, fees)
            cash_transactions = root.findall(".//CashTransaction")
            for cash_txn in cash_transactions:
                transaction = self._parse_cash_transaction(cash_txn)
                if transaction:
                    transactions.append(transaction)

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

    def _parse_trade(self, trade_element: ET.Element) -> Optional[Dict]:
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
            quantity = float(trade_element.get("quantity", 0))
            price = float(trade_element.get("tradePrice", 0))
            amount = float(trade_element.get("netCash", 0))  # Negative for purchases
            # IBKR may use either 'currency' or 'currencyPrimary'
            currency = trade_element.get("currencyPrimary") or trade_element.get("currency", "USD")
            commission = abs(float(trade_element.get("ibCommission", 0)))

            # Generate unique transaction ID
            ibkr_transaction_id = (
                f"{trade_element.get('transactionID', '')}_{trade_element.get('ibOrderID', '')}"
            )

            # Determine transaction type
            if quantity > 0:
                transaction_type = "buy"
            elif quantity < 0:
                transaction_type = "sell"
                quantity = abs(quantity)
            else:
                return None

            # Parse date
            transaction_date = datetime.strptime(date_str, "%Y%m%d").date()

            return {
                "ibkr_transaction_id": ibkr_transaction_id,
                "transaction_date": transaction_date,
                "symbol": symbol,
                "isin": isin,
                "description": description,
                "transaction_type": transaction_type,
                "quantity": quantity,
                "price": abs(price),
                "total_amount": abs(amount),
                "currency": currency,
                "fees": commission,
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

    def _parse_cash_transaction(self, cash_element: ET.Element) -> Optional[Dict]:
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
            date_str = cash_element.get("reportDate") or cash_element.get("dateTime", "")
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

            # Parse date (handle both date formats)
            if ";" in date_str:
                date_str = date_str.split(";")[0]
            if len(date_str) == 8:
                transaction_date = datetime.strptime(date_str, "%Y%m%d").date()
            else:
                transaction_date = datetime.strptime(date_str.split(",")[0], "%Y-%m-%d").date()

            # Enhance description with additional details
            enhanced_description = description
            if code:
                enhanced_description += f" [Code: {code}]"
            if ex_date and transaction_type == "dividend":
                enhanced_description += f" [Ex-Date: {ex_date}]"

            return {
                "ibkr_transaction_id": ibkr_transaction_id,
                "transaction_date": transaction_date,
                "symbol": symbol,
                "isin": isin,
                "description": enhanced_description,
                "transaction_type": transaction_type,
                "quantity": None,  # No quantity for cash transactions
                "price": None,
                "total_amount": abs(amount),
                "currency": currency,
                "fees": 0.0,
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

    def import_transactions(self, transactions: List[Dict]) -> Dict:
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

    def test_connection(self, token: str, query_id: str) -> Dict:
        """
        Test IBKR Flex connection.

        Args:
            token: IBKR Flex token
            query_id: IBKR Flex query ID

        Returns:
            Dictionary with test results
        """
        try:
            # Try to fetch statement (don't cache test requests)
            xml_data = self.fetch_statement(token, query_id, use_cache=False)

            if xml_data:
                # Try to parse it
                transactions = self.parse_flex_statement(xml_data)
                return {
                    "success": True,
                    "message": "Connection successful",
                    "transaction_count": len(transactions),
                }
            else:
                return {
                    "success": False,
                    "message": "Failed to fetch statement from IBKR",
                }

        except Exception as e:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Error testing IBKR connection",
                details={"error": str(e)},
            )
            return {"success": False, "message": f"Connection test failed: {str(e)}"}
