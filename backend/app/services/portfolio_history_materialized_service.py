"""
Service for managing portfolio history materialized views.

This service handles the creation, querying, and invalidation of pre-calculated
fund-level history data stored in the fund_history_materialized table.

The fund-level data is aggregated on-the-fly to provide portfolio-level views,
eliminating data duplication while maintaining query flexibility.
"""

from datetime import date, datetime

from sqlalchemy import func

from ..models import (
    FundHistoryMaterialized,
    LogCategory,
    LogLevel,
    Portfolio,
    PortfolioFund,
    RealizedGainLoss,
    db,
)
from ..services.logging_service import logger
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
    - Querying materialized history (aggregated from fund-level data)
    - Materializing new data at fund level
    - Invalidating cached data
    """

    @staticmethod
    def check_materialized_coverage(
        portfolio_ids: list[str], start_date: date, end_date: date
    ) -> MaterializedCoverage:
        """
        Check if the requested date range is fully materialized for the given portfolios.

        Now checks fund-level data coverage instead of portfolio-level.

        Args:
            portfolio_ids: List of portfolio IDs to check
            start_date: Start date of requested range
            end_date: End date of requested range

        Returns:
            MaterializedCoverage object with coverage information
        """
        if not portfolio_ids:
            return MaterializedCoverage(is_complete=True)

        # Get all portfolio_fund IDs for these portfolios
        portfolio_fund_ids = (
            db.session.query(PortfolioFund.id)
            .filter(PortfolioFund.portfolio_id.in_(portfolio_ids))
            .all()
        )
        portfolio_fund_ids = [pf[0] for pf in portfolio_fund_ids]

        if not portfolio_fund_ids:
            # No funds in these portfolios, consider it complete (nothing to materialize)
            return MaterializedCoverage(is_complete=True)

        # Query for existing materialized records in the date range
        # We need at least one record per portfolio_fund per day
        existing_records = (
            db.session.query(
                FundHistoryMaterialized.portfolio_fund_id,
                FundHistoryMaterialized.date,
            )
            .filter(
                FundHistoryMaterialized.portfolio_fund_id.in_(portfolio_fund_ids),
                FundHistoryMaterialized.date >= start_date.isoformat(),
                FundHistoryMaterialized.date <= end_date.isoformat(),
            )
            .all()
        )

        # Calculate expected number of records (days * portfolio_funds)
        days_in_range = (end_date - start_date).days + 1
        expected_total = days_in_range * len(portfolio_fund_ids)
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
        Get materialized portfolio history by aggregating fund-level data.

        Args:
            portfolio_ids: List of portfolio IDs
            start_date: Start date
            end_date: End date

        Returns:
            List of daily portfolio values in the same format as get_portfolio_history
        """
        # Aggregate fund-level data to portfolio level using SQL
        records = (
            db.session.query(
                FundHistoryMaterialized.date,
                PortfolioFund.portfolio_id,
                func.sum(FundHistoryMaterialized.value).label("total_value"),
                func.sum(FundHistoryMaterialized.cost).label("total_cost"),
                func.sum(FundHistoryMaterialized.realized_gain).label("total_realized_gain"),
                func.sum(FundHistoryMaterialized.unrealized_gain).label("total_unrealized_gain"),
                func.sum(FundHistoryMaterialized.total_gain_loss).label("total_gain_loss"),
                func.sum(FundHistoryMaterialized.dividends).label("total_dividends"),
                func.sum(FundHistoryMaterialized.fees).label("total_fees"),
            )
            .join(PortfolioFund, FundHistoryMaterialized.portfolio_fund_id == PortfolioFund.id)
            .filter(
                PortfolioFund.portfolio_id.in_(portfolio_ids),
                FundHistoryMaterialized.date >= start_date.isoformat(),
                FundHistoryMaterialized.date <= end_date.isoformat(),
            )
            .group_by(FundHistoryMaterialized.date, PortfolioFund.portfolio_id)
            .order_by(FundHistoryMaterialized.date.asc())
            .all()
        )

        # Get portfolio metadata
        portfolios = {
            p.id: p for p in Portfolio.query.filter(Portfolio.id.in_(portfolio_ids)).all()
        }

        # Get realized gain details for sale proceeds and original cost
        # (these are not stored in fund_history_materialized)
        realized_gain_records = RealizedGainLoss.query.filter(
            RealizedGainLoss.portfolio_id.in_(portfolio_ids),
            RealizedGainLoss.transaction_date >= start_date,
            RealizedGainLoss.transaction_date <= end_date,
        ).all()

        # Build lookup for realized gains by portfolio and date
        realized_by_portfolio_date = {}
        for rg in realized_gain_records:
            key = (
                rg.portfolio_id,
                rg.transaction_date.isoformat()
                if hasattr(rg.transaction_date, "isoformat")
                else str(rg.transaction_date),
            )
            if key not in realized_by_portfolio_date:
                realized_by_portfolio_date[key] = {"sale_proceeds": 0, "original_cost": 0}
            realized_by_portfolio_date[key]["sale_proceeds"] += rg.sale_proceeds
            realized_by_portfolio_date[key]["original_cost"] += rg.cost_basis

        # Build cumulative totals for sale proceeds and original cost
        cumulative_by_portfolio = {
            pid: {"sale_proceeds": 0, "original_cost": 0} for pid in portfolio_ids
        }

        # Group by date
        history_by_date = {}
        for record in records:
            date_str = record.date
            portfolio_id = record.portfolio_id

            if date_str not in history_by_date:
                history_by_date[date_str] = {"date": date_str, "portfolios": []}

            portfolio = portfolios.get(portfolio_id)
            if not portfolio:
                continue

            # Add any realized gains from this date to cumulative totals
            key = (portfolio_id, date_str)
            if key in realized_by_portfolio_date:
                cumulative_by_portfolio[portfolio_id]["sale_proceeds"] += (
                    realized_by_portfolio_date[key]["sale_proceeds"]
                )
                cumulative_by_portfolio[portfolio_id]["original_cost"] += (
                    realized_by_portfolio_date[key]["original_cost"]
                )

            history_by_date[date_str]["portfolios"].append(
                {
                    "id": portfolio_id,
                    "name": portfolio.name,
                    "totalValue": round(record.total_value or 0, 6),
                    "totalCost": round(record.total_cost or 0, 6),
                    "totalRealizedGainLoss": round(record.total_realized_gain or 0, 6),
                    "totalUnrealizedGainLoss": round(record.total_unrealized_gain or 0, 6),
                    "totalDividends": round(record.total_dividends or 0, 6),
                    "totalSaleProceeds": round(
                        cumulative_by_portfolio[portfolio_id]["sale_proceeds"], 6
                    ),
                    "totalOriginalCost": round(
                        cumulative_by_portfolio[portfolio_id]["original_cost"], 6
                    ),
                    "totalGainLoss": round(record.total_gain_loss or 0, 6),
                    "isArchived": portfolio.is_archived,
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
        Calculate and store fund-level history in the materialized view.

        Args:
            portfolio_id: Portfolio ID to materialize
            start_date: Start date (defaults to first transaction)
            end_date: End date (defaults to today)
            force_recalculate: If True, recalculate even if data exists

        Returns:
            Number of fund records materialized
        """
        from ..models import Transaction

        # Get portfolio
        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError(f"Portfolio {portfolio_id} not found")

        # Get portfolio funds
        portfolio_funds = PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
        if not portfolio_funds:
            return 0

        portfolio_fund_ids = [pf.id for pf in portfolio_funds]

        # Determine date range
        if start_date is None:
            # Find first transaction date
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

        # If force recalculate, delete existing records in this range
        if force_recalculate:
            FundHistoryMaterialized.query.filter(
                FundHistoryMaterialized.portfolio_fund_id.in_(portfolio_fund_ids),
                FundHistoryMaterialized.date >= start_date.isoformat(),
                FundHistoryMaterialized.date <= end_date.isoformat(),
            ).delete(synchronize_session=False)
            db.session.commit()

        # Calculate fund history using existing service
        fund_history = PortfolioService.get_portfolio_fund_history(
            portfolio_id,
            start_date=start_date.isoformat(),
            end_date=end_date.isoformat(),
        )

        # Get realized gains for each fund
        realized_gains_by_fund = {}
        realized_records = RealizedGainLoss.query.filter_by(portfolio_id=portfolio_id).all()
        for rg in realized_records:
            if rg.fund_id not in realized_gains_by_fund:
                realized_gains_by_fund[rg.fund_id] = []
            realized_gains_by_fund[rg.fund_id].append(rg)

        # Get dividends for each portfolio fund
        from ..models import Dividend

        dividends_by_pf = {}
        dividends = Dividend.query.filter(Dividend.portfolio_fund_id.in_(portfolio_fund_ids)).all()
        for d in dividends:
            if d.portfolio_fund_id not in dividends_by_pf:
                dividends_by_pf[d.portfolio_fund_id] = []
            dividends_by_pf[d.portfolio_fund_id].append(d)

        # Insert fund-level records
        records_inserted = 0
        for daily_data in fund_history:
            date_str = daily_data["date"]

            for fund_data in daily_data.get("funds", []):
                portfolio_fund_id = fund_data["portfolioFundId"]
                fund_id = fund_data["fundId"]

                # Check if record already exists
                existing = FundHistoryMaterialized.query.filter_by(
                    portfolio_fund_id=portfolio_fund_id, date=date_str
                ).first()

                if existing and not force_recalculate:
                    continue

                # Calculate cumulative realized gains for this fund up to this date
                realized_gain = 0
                if fund_id in realized_gains_by_fund:
                    for rg in realized_gains_by_fund[fund_id]:
                        rg_date = (
                            rg.transaction_date.isoformat()
                            if hasattr(rg.transaction_date, "isoformat")
                            else str(rg.transaction_date)
                        )
                        if rg_date <= date_str:
                            realized_gain += rg.realized_gain_loss

                # Calculate cumulative dividends for this fund up to this date
                total_dividends = 0
                if portfolio_fund_id in dividends_by_pf:
                    for d in dividends_by_pf[portfolio_fund_id]:
                        d_date = (
                            d.ex_dividend_date.isoformat()
                            if hasattr(d.ex_dividend_date, "isoformat")
                            else str(d.ex_dividend_date)
                        )
                        if d_date <= date_str:
                            total_dividends += d.total_amount

                # Calculate unrealized gain
                unrealized_gain = fund_data.get(
                    "unrealizedGain", fund_data["value"] - fund_data["cost"]
                )

                # Calculate total gain/loss
                total_gain_loss = realized_gain + unrealized_gain

                if existing:
                    # Update existing record
                    existing.fund_id = fund_id
                    existing.shares = fund_data["shares"]
                    existing.price = fund_data["price"]
                    existing.value = fund_data["value"]
                    existing.cost = fund_data["cost"]
                    existing.realized_gain = realized_gain
                    existing.unrealized_gain = unrealized_gain
                    existing.total_gain_loss = total_gain_loss
                    existing.dividends = total_dividends
                    existing.fees = 0  # Fees can be added later if needed
                    existing.calculated_at = datetime.now()
                else:
                    # Create new record
                    record = FundHistoryMaterialized(
                        portfolio_fund_id=portfolio_fund_id,
                        fund_id=fund_id,
                        date=date_str,
                        shares=fund_data["shares"],
                        price=fund_data["price"],
                        value=fund_data["value"],
                        cost=fund_data["cost"],
                        realized_gain=realized_gain,
                        unrealized_gain=unrealized_gain,
                        total_gain_loss=total_gain_loss,
                        dividends=total_dividends,
                        fees=0,
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
        # Get portfolio fund IDs for this portfolio
        portfolio_fund_ids = [
            pf.id for pf in PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
        ]

        if not portfolio_fund_ids:
            return 0

        deleted = FundHistoryMaterialized.query.filter(
            FundHistoryMaterialized.portfolio_fund_id.in_(portfolio_fund_ids),
            FundHistoryMaterialized.date >= from_date.isoformat(),
        ).delete(synchronize_session=False)

        db.session.commit()

        # Log invalidation results
        logger.log(
            level=LogLevel.INFO if deleted > 0 else LogLevel.DEBUG,
            category=LogCategory.SYSTEM,
            message=f"Materialized view invalidation for portfolio {portfolio_id}",
            details={
                "portfolio_id": portfolio_id,
                "from_date": from_date.isoformat(),
                "records_deleted": deleted,
                "portfolio_funds_checked": len(portfolio_fund_ids),
                "recalculate": recalculate,
            },
        )

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
        total_records = FundHistoryMaterialized.query.count()

        # Get unique portfolio funds with data
        portfolio_funds_with_data = (
            db.session.query(FundHistoryMaterialized.portfolio_fund_id).distinct().count()
        )

        # Get unique portfolios with data
        portfolios_with_data = (
            db.session.query(PortfolioFund.portfolio_id)
            .join(
                FundHistoryMaterialized,
                PortfolioFund.id == FundHistoryMaterialized.portfolio_fund_id,
            )
            .distinct()
            .count()
        )

        # Get date range
        if total_records > 0:
            oldest = (
                FundHistoryMaterialized.query.order_by(FundHistoryMaterialized.date.asc())
                .first()
                .date
            )
            newest = (
                FundHistoryMaterialized.query.order_by(FundHistoryMaterialized.date.desc())
                .first()
                .date
            )
        else:
            oldest = None
            newest = None

        return {
            "total_records": total_records,
            "portfolio_funds_with_data": portfolio_funds_with_data,
            "portfolios_with_data": portfolios_with_data,
            "oldest_date": oldest,
            "newest_date": newest,
        }
