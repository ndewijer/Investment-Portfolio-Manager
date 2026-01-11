"""
Service for managing portfolio history materialized views.

This service handles the creation, querying, and invalidation of pre-calculated
portfolio history data stored in the portfolio_history_materialized table.
"""

from datetime import date, datetime

from ..models import (
    Portfolio,
    PortfolioHistoryMaterialized,
    db,
)
from .portfolio_service import PortfolioService


class MaterializedCoverage:
    """
    Represents coverage information for materialized portfolio history.

    Attributes:
        is_complete (bool): Whether the requested date range is fully materialized
        partial_coverage (bool): Whether there is partial coverage
        missing_ranges (List[Tuple[date, date]]): List of date ranges that are not materialized
        covered_ranges (List[Tuple[date, date]]): List of date ranges that are materialized
    """

    def __init__(
        self,
        is_complete: bool = False,
        partial_coverage: bool = False,
        missing_ranges: list[tuple[date, date]] | None = None,
        covered_ranges: list[tuple[date, date]] | None = None,
    ):
        self.is_complete = is_complete
        self.partial_coverage = partial_coverage
        self.missing_ranges = missing_ranges or []
        self.covered_ranges = covered_ranges or []


class PortfolioHistoryMaterializedService:
    """
    Service for managing materialized portfolio history.

    Provides methods for:
    - Checking coverage of materialized data
    - Querying materialized history
    - Materializing new data
    - Invalidating cached data
    """

    @staticmethod
    def check_materialized_coverage(
        portfolio_ids: list[str], start_date: date, end_date: date
    ) -> MaterializedCoverage:
        """
        Check if the requested date range is fully materialized for the given portfolios.

        Args:
            portfolio_ids: List of portfolio IDs to check
            start_date: Start date of requested range
            end_date: End date of requested range

        Returns:
            MaterializedCoverage object with coverage information
        """
        if not portfolio_ids:
            return MaterializedCoverage(is_complete=True)

        # Query for existing materialized records in the date range
        existing_records = (
            db.session.query(
                PortfolioHistoryMaterialized.portfolio_id,
                PortfolioHistoryMaterialized.date,
            )
            .filter(
                PortfolioHistoryMaterialized.portfolio_id.in_(portfolio_ids),
                PortfolioHistoryMaterialized.date >= start_date.isoformat(),
                PortfolioHistoryMaterialized.date <= end_date.isoformat(),
            )
            .all()
        )

        # Calculate expected number of records (days * portfolios)
        days_in_range = (end_date - start_date).days + 1
        expected_total = days_in_range * len(portfolio_ids)
        actual_total = len(existing_records)

        if actual_total == 0:
            return MaterializedCoverage(
                is_complete=False,
                partial_coverage=False,
                missing_ranges=[(start_date, end_date)],
            )

        if actual_total == expected_total:
            return MaterializedCoverage(
                is_complete=True,
                partial_coverage=False,
                covered_ranges=[(start_date, end_date)],
            )

        # Partial coverage - would need more sophisticated gap analysis
        # For now, we'll treat partial as incomplete and recalculate
        return MaterializedCoverage(
            is_complete=False,
            partial_coverage=True,
            missing_ranges=[(start_date, end_date)],
        )

    @staticmethod
    def get_materialized_history(
        portfolio_ids: list[str], start_date: date, end_date: date
    ) -> list[dict]:
        """
        Get materialized portfolio history from the cache.

        Args:
            portfolio_ids: List of portfolio IDs
            start_date: Start date
            end_date: End date

        Returns:
            List of daily portfolio values in the same format as get_portfolio_history
        """
        # Query materialized data
        records = (
            PortfolioHistoryMaterialized.query.filter(
                PortfolioHistoryMaterialized.portfolio_id.in_(portfolio_ids),
                PortfolioHistoryMaterialized.date >= start_date.isoformat(),
                PortfolioHistoryMaterialized.date <= end_date.isoformat(),
            )
            .order_by(PortfolioHistoryMaterialized.date.asc())
            .all()
        )

        # Group by date
        history_by_date = {}
        for record in records:
            if record.date not in history_by_date:
                history_by_date[record.date] = {"date": record.date, "portfolios": []}

            # Get portfolio name
            portfolio = db.session.get(Portfolio, record.portfolio_id)
            portfolio_name = portfolio.name if portfolio else "Unknown"

            history_by_date[record.date]["portfolios"].append(
                {
                    "id": record.portfolio_id,
                    "name": portfolio_name,
                    "value": round(record.value, 6),
                    "cost": round(record.cost, 6),
                    "realized_gain": round(record.realized_gain, 6),
                    "unrealized_gain": round(record.unrealized_gain, 6),
                    "total_dividends": round(record.total_dividends, 6),
                    "total_sale_proceeds": round(record.total_sale_proceeds, 6),
                    "total_original_cost": round(record.total_original_cost, 6),
                    "total_gain_loss": round(record.total_gain_loss, 6),
                    "is_archived": bool(record.is_archived),
                }
            )

        # Convert to list sorted by date
        return [history_by_date[d] for d in sorted(history_by_date.keys())]

    @staticmethod
    def materialize_portfolio_history(
        portfolio_id: str,
        start_date: date | None = None,
        end_date: date | None = None,
        force_recalculate: bool = False,
    ) -> int:
        """
        Calculate and store portfolio history in the materialized view.

        Args:
            portfolio_id: Portfolio ID to materialize
            start_date: Start date (defaults to first transaction)
            end_date: End date (defaults to today)
            force_recalculate: If True, recalculate even if data exists

        Returns:
            Number of records materialized
        """

        # Get portfolio
        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError(f"Portfolio {portfolio_id} not found")

        # Determine date range
        if start_date is None:
            # Find first transaction date
            from ..models import PortfolioFund, Transaction

            earliest_tx = (
                Transaction.query.join(PortfolioFund)
                .filter(PortfolioFund.portfolio_id == portfolio_id)
                .order_by(Transaction.date.asc())
                .first()
            )
            if not earliest_tx:
                return 0
            start_date = earliest_tx.date

        if end_date is None:
            end_date = datetime.now().date()

        # If not force recalculate, delete only missing ranges
        if force_recalculate:
            # Delete existing records in this range
            PortfolioHistoryMaterialized.query.filter(
                PortfolioHistoryMaterialized.portfolio_id == portfolio_id,
                PortfolioHistoryMaterialized.date >= start_date.isoformat(),
                PortfolioHistoryMaterialized.date <= end_date.isoformat(),
            ).delete(synchronize_session=False)
            db.session.commit()

        # Calculate history using existing service
        # We need to call it for this specific portfolio only

        # Temporarily filter to only this portfolio by calling the calculation
        # for just this portfolio's date range
        # Include hidden (exclude_from_overview=True) portfolios but not archived ones
        history = PortfolioService.get_portfolio_history(
            start_date=start_date.isoformat(), end_date=end_date.isoformat(), include_excluded=True
        )

        # Filter to only this portfolio's data and insert into materialized view
        records_inserted = 0
        for daily_data in history:
            date_str = daily_data["date"]
            for portfolio_data in daily_data["portfolios"]:
                if portfolio_data["id"] != portfolio_id:
                    continue

                # Check if record already exists
                existing = PortfolioHistoryMaterialized.query.filter_by(
                    portfolio_id=portfolio_id, date=date_str
                ).first()

                if existing and not force_recalculate:
                    continue

                if existing:
                    # Update existing record
                    existing.value = portfolio_data["value"]
                    existing.cost = portfolio_data["cost"]
                    existing.realized_gain = portfolio_data["realized_gain"]
                    existing.unrealized_gain = portfolio_data["unrealized_gain"]
                    existing.total_dividends = portfolio_data["total_dividends"]
                    existing.total_sale_proceeds = portfolio_data["total_sale_proceeds"]
                    existing.total_original_cost = portfolio_data["total_original_cost"]
                    existing.total_gain_loss = portfolio_data["total_gain_loss"]
                    existing.is_archived = int(portfolio_data["is_archived"])
                    existing.calculated_at = datetime.now()
                else:
                    # Create new record
                    record = PortfolioHistoryMaterialized(
                        portfolio_id=portfolio_id,
                        date=date_str,
                        value=portfolio_data["value"],
                        cost=portfolio_data["cost"],
                        realized_gain=portfolio_data["realized_gain"],
                        unrealized_gain=portfolio_data["unrealized_gain"],
                        total_dividends=portfolio_data["total_dividends"],
                        total_sale_proceeds=portfolio_data["total_sale_proceeds"],
                        total_original_cost=portfolio_data["total_original_cost"],
                        total_gain_loss=portfolio_data["total_gain_loss"],
                        is_archived=int(portfolio_data["is_archived"]),
                    )
                    db.session.add(record)

                records_inserted += 1

        db.session.commit()
        return records_inserted

    @staticmethod
    def invalidate_materialized_history(
        portfolio_id: str, from_date: date, recalculate: bool = False
    ) -> int:
        """
        Invalidate (delete) materialized history from a certain date forward.

        This should be called when source data changes (transactions, prices, etc.)
        to ensure the materialized view stays in sync.

        Args:
            portfolio_id: Portfolio ID
            from_date: Delete all records on or after this date
            recalculate: If True, immediately recalculate after invalidation

        Returns:
            Number of records deleted
        """
        deleted = PortfolioHistoryMaterialized.query.filter(
            PortfolioHistoryMaterialized.portfolio_id == portfolio_id,
            PortfolioHistoryMaterialized.date >= from_date.isoformat(),
        ).delete(synchronize_session=False)

        db.session.commit()

        if recalculate and deleted > 0:
            # Recalculate from the invalidated date to today
            PortfolioHistoryMaterializedService.materialize_portfolio_history(
                portfolio_id, start_date=from_date, end_date=datetime.now().date()
            )

        return deleted

    @staticmethod
    def materialize_all_portfolios(force_recalculate: bool = False) -> dict[str, int]:
        """
        Materialize history for all portfolios.

        Args:
            force_recalculate: If True, recalculate even if data exists

        Returns:
            Dictionary mapping portfolio_id to number of records materialized
        """
        portfolios = Portfolio.query.all()
        results = {}

        for portfolio in portfolios:
            try:
                count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
                    portfolio.id, force_recalculate=force_recalculate
                )
                results[portfolio.id] = count
            except Exception as e:
                results[portfolio.id] = f"Error: {e!s}"

        return results

    @staticmethod
    def invalidate_from_transaction(transaction) -> int:
        """
        Invalidate materialized history based on a transaction.

        Args:
            transaction: Transaction object

        Returns:
            Number of records deleted
        """
        # Get portfolio_id from the transaction
        portfolio_fund = transaction.portfolio_fund
        if not portfolio_fund:
            return 0

        portfolio_id = portfolio_fund.portfolio_id
        transaction_date = transaction.date

        return PortfolioHistoryMaterializedService.invalidate_materialized_history(
            portfolio_id, transaction_date, recalculate=False
        )

    @staticmethod
    def invalidate_from_dividend(dividend) -> int:
        """
        Invalidate materialized history based on a dividend.

        Args:
            dividend: Dividend object

        Returns:
            Number of records deleted
        """
        # Get portfolio_id from the dividend
        portfolio_fund_id = dividend.portfolio_fund_id
        from ..models import PortfolioFund

        portfolio_fund = db.session.get(PortfolioFund, portfolio_fund_id)
        if not portfolio_fund:
            return 0

        portfolio_id = portfolio_fund.portfolio_id
        dividend_date = dividend.ex_dividend_date

        return PortfolioHistoryMaterializedService.invalidate_materialized_history(
            portfolio_id, dividend_date, recalculate=False
        )

    @staticmethod
    def invalidate_from_price_update(fund_id, price_date) -> int:
        """
        Invalidate materialized history for all portfolios holding a fund.

        Args:
            fund_id: Fund ID that had a price update
            price_date: Date of the price update

        Returns:
            Total number of records deleted
        """
        from ..models import PortfolioFund

        # Find all portfolios holding this fund
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()

        total_deleted = 0
        for pf in portfolio_funds:
            deleted = PortfolioHistoryMaterializedService.invalidate_materialized_history(
                pf.portfolio_id, price_date, recalculate=False
            )
            total_deleted += deleted

        return total_deleted

    @staticmethod
    def get_materialized_stats() -> dict:
        """
        Get statistics about the materialized view.

        Returns:
            Dictionary with statistics
        """
        total_records = PortfolioHistoryMaterialized.query.count()
        portfolios_with_data = (
            db.session.query(PortfolioHistoryMaterialized.portfolio_id).distinct().count()
        )

        # Get date range
        if total_records > 0:
            oldest = (
                PortfolioHistoryMaterialized.query.order_by(PortfolioHistoryMaterialized.date.asc())
                .first()
                .date
            )
            newest = (
                PortfolioHistoryMaterialized.query.order_by(
                    PortfolioHistoryMaterialized.date.desc()
                )
                .first()
                .date
            )
        else:
            oldest = None
            newest = None

        return {
            "total_records": total_records,
            "portfolios_with_data": portfolios_with_data,
            "oldest_date": oldest,
            "newest_date": newest,
        }
