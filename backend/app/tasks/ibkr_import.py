"""Scheduled task to import IBKR transactions automatically."""

from datetime import UTC, datetime

from ..models import IBKRConfig, LogCategory, LogLevel
from ..services.ibkr_flex_service import IBKRFlexService
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

        # Update last import date
        from ..models import db

        config.last_import_date = datetime.now(UTC)
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.IBKR,
            message=(
                f"Automated IBKR import completed: {results['imported']} imported, "
                f"{results['skipped']} skipped, {len(results['errors'])} errors"
            ),
            details=results,
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
