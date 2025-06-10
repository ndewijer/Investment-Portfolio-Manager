"""
Service class for portfolio-related operations.

This module provides methods for calculating portfolio values, managing
portfolio funds, and retrieving portfolio history and summaries.
"""

from datetime import datetime, timedelta

from ..models import (
    Dividend,
    FundPrice,
    Portfolio,
    PortfolioFund,
    Transaction,
    RealizedGainLoss,
)


class PortfolioService:
    """
    Service class for portfolio-related operations.

    Provides methods for:
    - Calculating portfolio values and returns
    - Managing portfolio funds
    - Retrieving portfolio history and summaries
    """

    @staticmethod
    def _process_transactions_for_date(transactions, date, dividend_shares=0):
        """
        Process transactions up to a specific date to calculate shares and cost.

        Args:
            transactions (list): List of Transaction objects
            date (date): Date to process up to
            dividend_shares (float): Additional dividend shares to include

        Returns:
            tuple: (total_shares, total_cost)
        """
        shares = dividend_shares
        cost = 0

        for transaction in transactions:
            if transaction.date <= date:
                if transaction.type == "buy":
                    shares += transaction.shares
                    cost += transaction.shares * transaction.cost_per_share
                elif transaction.type == "sell":
                    shares -= transaction.shares
                    # Adjust cost proportionally to remaining shares
                    if shares > 0:
                        cost = (cost / (shares + transaction.shares)) * shares
                    else:
                        cost = 0

        return max(0, round(shares, 6)), max(0, cost)

    @staticmethod
    def _get_fund_price_for_date(fund_id, date):
        """
        Get the latest fund price on or before a specific date.

        Args:
            fund_id: Fund identifier
            date: Target date

        Returns:
            FundPrice object or None
        """
        return (
            FundPrice.query.filter(FundPrice.fund_id == fund_id, FundPrice.date <= date)
            .order_by(FundPrice.date.desc())
            .first()
        )

    @staticmethod
    def _get_dividend_shares_for_date(portfolio_fund_id, date, all_dividends=None):
        """
        Get total dividend shares for a portfolio fund up to a specific date.

        Args:
            portfolio_fund_id: Portfolio fund identifier
            date: Target date
            all_dividends: Pre-fetched dividend data (optional)

        Returns:
            float: Total dividend shares
        """
        if all_dividends and portfolio_fund_id in all_dividends:
            return sum(
                dividend["shares"]
                for dividend in all_dividends[portfolio_fund_id]
                if dividend["ex_dividend_date"] <= date
            )

        # Fallback to database query
        dividend_shares = (
            Dividend.query.filter_by(portfolio_fund_id=portfolio_fund_id)
            .filter(Dividend.ex_dividend_date <= date)
            .join(
                Transaction,
                Dividend.reinvestment_transaction_id == Transaction.id,
                isouter=True,
            )
            .with_entities(Transaction.shares)
            .all()
        )
        return sum(d.shares or 0 for d in dividend_shares)

    @staticmethod
    def _calculate_fund_metrics(
        portfolio_fund,
        target_date=None,
        all_dividends=None,
        force_historical_format=False,
    ):
        """
        Calculate metrics for a single portfolio fund.

        Args:
            portfolio_fund: PortfolioFund object
            target_date: Date to calculate for (None for current)
            all_dividends: Pre-fetched dividend data (optional)

        Returns:
            dict: Fund metrics including shares, cost, value, etc.
        """
        if target_date is None:
            target_date = datetime.now().date()

        # Get dividend shares
        dividend_shares = PortfolioService._get_dividend_shares_for_date(
            portfolio_fund.id, target_date, all_dividends
        )

        # Get transactions sorted by date
        transactions = (
            Transaction.query.filter_by(portfolio_fund_id=portfolio_fund.id)
            .filter(Transaction.date <= target_date)
            .order_by(Transaction.date.asc())
            .all()
        )

        # Process transactions
        shares, cost = PortfolioService._process_transactions_for_date(
            transactions, target_date, dividend_shares
        )

        # Get latest price
        latest_price = PortfolioService._get_fund_price_for_date(
            portfolio_fund.fund_id, target_date
        )
        price_value = latest_price.price if latest_price else 0

        # Calculate values
        current_value = shares * price_value

        # For current date calculations, include realized gains and dividends
        if target_date == datetime.now().date() and not force_historical_format:
            # Calculate realized gains/losses
            realized_records = RealizedGainLoss.query.filter_by(
                portfolio_id=portfolio_fund.portfolio_id, fund_id=portfolio_fund.fund_id
            ).all()
            realized_gain_loss = sum(r.realized_gain_loss for r in realized_records)

            # Calculate total dividends
            dividends = Dividend.query.filter_by(
                portfolio_fund_id=portfolio_fund.id
            ).all()
            total_dividends = sum(d.total_amount for d in dividends)

            return {
                "id": portfolio_fund.id,
                "fund_id": portfolio_fund.fund_id,
                "fund_name": portfolio_fund.fund.name,
                "total_shares": shares,
                "latest_price": price_value,
                "average_cost": cost / shares if shares > 0 else 0,
                "total_cost": cost,
                "current_value": current_value,
                "unrealized_gain_loss": current_value - cost,
                "realized_gain_loss": realized_gain_loss,
                "total_gain_loss": realized_gain_loss + (current_value - cost),
                "total_dividends": total_dividends,
                "dividend_type": portfolio_fund.fund.dividend_type.value,
            }
        else:
            # Historical calculation - simpler return
            return {
                "fund_id": portfolio_fund.fund_id,
                "fund_name": portfolio_fund.fund.name,
                "shares": shares,
                "cost": cost,
                "value": current_value,
                "price": price_value,
            }

    @staticmethod
    def _format_portfolio_summary(portfolio, portfolio_funds_data):
        """
        Format portfolio data for summary response.

        Args:
            portfolio: Portfolio object
            portfolio_funds_data: List of fund metrics

        Returns:
            dict: Formatted portfolio summary
        """
        # Get realized gains records for additional metrics
        realized_gains = RealizedGainLoss.query.filter_by(
            portfolio_id=portfolio.id
        ).all()

        totals = {
            "totalValue": sum(pf["current_value"] for pf in portfolio_funds_data),
            "totalCost": sum(pf["total_cost"] for pf in portfolio_funds_data),
            "totalDividends": sum(pf["total_dividends"] for pf in portfolio_funds_data),
            "totalUnrealizedGainLoss": sum(
                pf["unrealized_gain_loss"] for pf in portfolio_funds_data
            ),
            "totalRealizedGainLoss": sum(
                pf["realized_gain_loss"] for pf in portfolio_funds_data
            ),
            "totalSaleProceeds": sum(gain.sale_proceeds for gain in realized_gains),
            "totalOriginalCost": sum(gain.cost_basis for gain in realized_gains),
            "totalGainLoss": sum(pf["total_gain_loss"] for pf in portfolio_funds_data),
        }

        return {
            "id": portfolio.id,
            "name": portfolio.name,
            **totals,
            "is_archived": portfolio.is_archived,
        }

    @staticmethod
    def _preload_dividend_data(portfolios):
        """
        Preload dividend data for multiple portfolios to avoid N+1 queries.

        Args:
            portfolios: List of Portfolio objects

        Returns:
            dict: Dividend data grouped by portfolio_fund_id
        """
        all_dividends = {}

        for portfolio in portfolios:
            portfolio_fund_ids = [pf.id for pf in portfolio.funds]
            if portfolio_fund_ids:
                dividends = (
                    Dividend.query.filter(
                        Dividend.portfolio_fund_id.in_(portfolio_fund_ids)
                    )
                    .join(
                        Transaction,
                        Dividend.reinvestment_transaction_id == Transaction.id,
                        isouter=True,
                    )
                    .with_entities(
                        Dividend.portfolio_fund_id,
                        Dividend.ex_dividend_date,
                        Transaction.shares,
                    )
                    .all()
                )

                # Group dividends by portfolio_fund_id
                for dividend in dividends:
                    pf_id = dividend.portfolio_fund_id
                    if pf_id not in all_dividends:
                        all_dividends[pf_id] = []
                    all_dividends[pf_id].append(
                        {
                            "ex_dividend_date": dividend.ex_dividend_date,
                            "shares": dividend.shares or 0,
                        }
                    )

        return all_dividends

    @staticmethod
    def calculate_portfolio_fund_values(portfolio_funds):
        """
        Calculate current values and metrics for a list of portfolio funds.

        Args:
            portfolio_funds (list): List of PortfolioFund objects

        Returns:
            list: List of dictionaries containing fund metrics
        """
        return [PortfolioService._calculate_fund_metrics(pf) for pf in portfolio_funds]

    @staticmethod
    def get_portfolio_summary():
        """
        Get summary of all non-archived and visible portfolios.

        Returns:
            list: List of portfolio summary dictionaries
        """
        portfolios = Portfolio.query.filter_by(
            is_archived=False, exclude_from_overview=False
        ).all()

        summary = []
        for portfolio in portfolios:
            portfolio_funds_data = PortfolioService.calculate_portfolio_fund_values(
                portfolio.funds
            )

            if portfolio_funds_data:
                has_transactions = any(
                    pf["total_shares"] > 0 for pf in portfolio_funds_data
                )

                if has_transactions:
                    summary.append(
                        PortfolioService._format_portfolio_summary(
                            portfolio, portfolio_funds_data
                        )
                    )

        return summary

    @staticmethod
    def get_portfolio_funds(portfolio_id):
        """
        Get portfolio funds for a specific portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            list: List of dictionaries containing portfolio fund details
        """
        portfolio_funds = PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
        return PortfolioService.calculate_portfolio_fund_values(portfolio_funds)

    @staticmethod
    def get_all_portfolio_funds():
        """
        Get all portfolio funds.

        Returns:
            list: List of dictionaries containing portfolio fund details
        """
        portfolio_funds = PortfolioFund.query.all()
        return [
            {
                "id": pf.id,
                "portfolio_id": pf.portfolio_id,
                "fund_id": pf.fund_id,
                "portfolio_name": pf.portfolio.name,
                "fund_name": pf.fund.name,
            }
            for pf in portfolio_funds
        ]

    @staticmethod
    def get_portfolio_history():
        """
        Get historical value data for all non-archived and visible portfolios.

        Returns:
            list: List of daily values containing portfolio history
        """
        portfolios = Portfolio.query.filter_by(
            is_archived=False, exclude_from_overview=False
        ).all()

        if not portfolios:
            return []

        # Get portfolio date ranges
        portfolio_date_ranges = {}
        for portfolio in portfolios:
            portfolio_transactions = (
                Transaction.query.join(PortfolioFund)
                .filter(PortfolioFund.portfolio_id == portfolio.id)
                .order_by(Transaction.date)
                .all()
            )

            if portfolio_transactions:
                portfolio_date_ranges[portfolio.id] = {
                    "start_date": portfolio_transactions[0].date,
                    "end_date": datetime.now().date(),
                }

        if not portfolio_date_ranges:
            return []

        # Preload dividend data
        all_dividends = PortfolioService._preload_dividend_data(portfolios)

        # Calculate date range
        one_year_ago = datetime.now().date() - timedelta(days=365)
        start_date = max(
            one_year_ago,
            min(range["start_date"] for range in portfolio_date_ranges.values()),
        )

        # Generate history
        history = []
        current_date = start_date
        today = datetime.now().date()

        while current_date <= today:
            daily_values = PortfolioService._calculate_daily_values(
                current_date, portfolios, portfolio_date_ranges, all_dividends
            )
            history.append(daily_values)
            current_date += timedelta(days=1)

        return history

    @staticmethod
    def _calculate_daily_values(
        current_date, portfolios, portfolio_date_ranges, all_dividends
    ):
        """Calculate portfolio values for a specific date."""
        daily_values = {"date": current_date.isoformat(), "portfolios": []}

        for portfolio in portfolios:
            if (
                portfolio.id not in portfolio_date_ranges
                or current_date < portfolio_date_ranges[portfolio.id]["start_date"]
            ):
                continue

            total_value = 0
            total_cost = 0

            for pf in portfolio.funds:
                fund_metrics = PortfolioService._calculate_fund_metrics(
                    pf, current_date, all_dividends, force_historical_format=True
                )
                try:
                    total_value += fund_metrics["value"]
                    total_cost += fund_metrics["cost"]
                except Exception as e:
                    print(f"Error processing fund metrics: {e}")

            if total_value > 0 or total_cost > 0:
                daily_values["portfolios"].append(
                    {
                        "id": portfolio.id,
                        "name": portfolio.name,
                        "value": total_value,
                        "cost": total_cost,
                    }
                )

        return daily_values

    @staticmethod
    def get_portfolio_fund_history(portfolio_id):
        """
        Get historical value data for funds in a portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            list: List of daily fund values
        """
        portfolio = Portfolio.query.get_or_404(portfolio_id)

        # Get the earliest transaction date for this portfolio
        earliest_transaction = (
            Transaction.query.join(PortfolioFund)
            .filter(PortfolioFund.portfolio_id == portfolio_id)
            .order_by(Transaction.date)
            .first()
        )

        if not earliest_transaction:
            return []

        history = []
        current_date = earliest_transaction.date
        today = datetime.now().date()

        while current_date <= today:
            daily_values = {"date": current_date.isoformat(), "funds": []}

            for pf in portfolio.funds:
                fund_metrics = PortfolioService._calculate_fund_metrics(
                    pf, current_date, force_historical_format=True
                )

                if fund_metrics["shares"] > 0:
                    daily_values["funds"].append(
                        {
                            "fund_id": fund_metrics["fund_id"],
                            "fund_name": fund_metrics["fund_name"],
                            "value": fund_metrics["value"],
                            "cost": fund_metrics["cost"],
                            "shares": fund_metrics["shares"],
                            "price": fund_metrics["price"],
                        }
                    )

            if daily_values["funds"]:
                history.append(daily_values)

            current_date += timedelta(days=1)

        return history
