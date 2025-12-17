"""Scheduled task to import IBKR transactions automatically."""

import json
from datetime import UTC, datetime

from ..models import IBKRConfig, IBKRTransaction, LogCategory, LogLevel
from ..services.ibkr_flex_service import IBKRFlexService
from ..services.ibkr_transaction_service import IBKRTransactionService
from ..services.logging_service import logger


def run_automated_ibkr_import():
    """
    Run automated IBKR import if enabled.

    This function:
    1. Checks if auto-import is enabled
    2. Fetches transactions from IBKR
    3. Imports transactions to inbox
    4. Updates last import timestamp
    """
    try:
        # Get IBKR configuration
        config = IBKRConfig.query.first()

        if not config:
            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message="IBKR not configured, skipping automated import",
                details={},
            )
            return

        if not config.enabled:
            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message="IBKR integration disabled, skipping automated import",
                details={},
            )
            return

        if not config.auto_import_enabled:
            logger.log(
                level=LogLevel.DEBUG,
                category=LogCategory.IBKR,
                message="IBKR auto-import disabled, skipping automated import",
                details={},
            )
            return

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message="Starting automated IBKR import",
            details={"query_id": config.flex_query_id},
        )

        # Initialize service
        service = IBKRFlexService()

        # Decrypt token
        token = service._decrypt_token(config.flex_token)

        # Fetch statement (use cache if available)
        xml_data = service.fetch_statement(token, config.flex_query_id, use_cache=True)

        if not xml_data:
            logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.IBKR,
                message="Failed to fetch IBKR statement during automated import",
                details={},
            )
            return

        # Parse transactions
        transactions = service.parse_flex_statement(xml_data)

        # Import transactions
        results = service.import_transactions(transactions)

        # Apply default allocations if enabled
        auto_allocated_count = 0
        auto_allocation_errors = []

        if config.default_allocation_enabled and config.default_allocations:
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.IBKR,
                message="Applying default allocations to newly imported transactions",
                details={"imported_count": results["imported"]},
            )

            try:
                # Parse default allocations
                allocations = json.loads(config.default_allocations)

                # Get all pending transactions (includes newly imported ones)
                pending_transactions = IBKRTransaction.query.filter_by(status="pending").all()

                # Try to allocate each pending transaction
                for transaction in pending_transactions:
                    try:
                        # Use the transaction service to process the allocation
                        # This handles all validation, fund matching, and transaction creation
                        result = IBKRTransactionService.process_transaction_allocation(
                            transaction.id, allocations
                        )

                        if result.get("success"):
                            auto_allocated_count += 1
                            logger.log(
                                level=LogLevel.DEBUG,
                                category=LogCategory.IBKR,
                                message=f"Auto-allocated transaction {transaction.symbol}",
                                details={
                                    "transaction_id": transaction.id,
                                    "symbol": transaction.symbol,
                                    "amount": transaction.total_amount,
                                },
                            )
                    except Exception as e:
                        # Log error but continue with other transactions
                        error_msg = (
                            f"Failed to auto-allocate transaction {transaction.symbol}: {e!s}"
                        )
                        auto_allocation_errors.append(error_msg)
                        logger.log(
                            level=LogLevel.WARNING,
                            category=LogCategory.IBKR,
                            message=error_msg,
                            details={
                                "transaction_id": transaction.id,
                                "symbol": transaction.symbol,
                                "error": str(e),
                            },
                        )

                if auto_allocated_count > 0:
                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.IBKR,
                        message=(
                            f"Auto-allocated {auto_allocated_count} transaction(s) "
                            f"using default preset"
                        ),
                        details={
                            "allocated": auto_allocated_count,
                            "errors": len(auto_allocation_errors),
                        },
                    )

            except json.JSONDecodeError as e:
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message="Invalid default allocations JSON",
                    details={"error": str(e)},
                )
            except Exception as e:
                logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.IBKR,
                    message="Error applying default allocations",
                    details={"error": str(e)},
                )

        # Update last import date
        from ..models import db

        config.last_import_date = datetime.now(UTC)
        db.session.commit()

        # Build summary message
        summary_parts = [
            f"{results['imported']} imported",
            f"{results['skipped']} skipped",
            f"{len(results['errors'])} errors",
        ]
        if auto_allocated_count > 0:
            summary_parts.append(f"{auto_allocated_count} auto-allocated")

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=f"Automated IBKR import completed: {', '.join(summary_parts)}",
            details={
                **results,
                "auto_allocated": auto_allocated_count,
                "auto_allocation_errors": len(auto_allocation_errors),
            },
        )

        if results["errors"]:
            logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.IBKR,
                message=f"Encountered {len(results['errors'])} errors during automated IBKR import",
                details={"errors": results["errors"]},
            )

    except Exception as e:
        logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.IBKR,
            message="Error during automated IBKR import",
            details={"error": str(e)},
        )


if __name__ == "__main__":
    run_automated_ibkr_import()
