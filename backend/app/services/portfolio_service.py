"""
Service class for portfolio-related operations.

This module provides methods for calculating portfolio values, managing
portfolio funds, and retrieving portfolio history and summaries.
"""

from collections import defaultdict
from datetime import datetime, timedelta

from sqlalchemy.orm import selectinload

from ..models import (
    Dividend,
    FundPrice,
    Portfolio,
    PortfolioFund,
    RealizedGainLoss,
    Transaction,
    db,
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
            force_historical_format: Whether to force the historical format output
                even for current date

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
            dividends = Dividend.query.filter_by(portfolio_fund_id=portfolio_fund.id).all()
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
                "investment_type": portfolio_fund.fund.investment_type.value,
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
        realized_gains = RealizedGainLoss.query.filter_by(portfolio_id=portfolio.id).all()

        totals = {
            "totalValue": sum(pf["current_value"] for pf in portfolio_funds_data),
            "totalCost": sum(pf["total_cost"] for pf in portfolio_funds_data),
            "totalDividends": sum(pf["total_dividends"] for pf in portfolio_funds_data),
            "totalUnrealizedGainLoss": sum(
                pf["unrealized_gain_loss"] for pf in portfolio_funds_data
            ),
            "totalRealizedGainLoss": sum(pf["realized_gain_loss"] for pf in portfolio_funds_data),
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
                    Dividend.query.filter(Dividend.portfolio_fund_id.in_(portfolio_fund_ids))
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
    def _load_historical_data_batch(portfolio_ids, start_date, end_date):
        """
        Load all historical data needed for calculations in a single batch.

        This method loads all transactions, prices, realized gains, and dividends
        for the specified portfolios within the date range, avoiding repeated
        database queries in loops.

        Args:
            portfolio_ids (list): List of portfolio IDs
            start_date (date): Start date for data loading
            end_date (date): End date for data loading

        Returns:
            dict: Dictionary containing:
                - transactions: List of all transactions
                - prices: List of all fund prices
                - realized_gains: List of all realized gain/loss records
                - dividends: List of all dividend records
        """
        # Load all transactions for these portfolios up to end_date
        transactions = (
            Transaction.query.join(PortfolioFund)
            .filter(
                PortfolioFund.portfolio_id.in_(portfolio_ids),
                Transaction.date <= end_date,
            )
            .order_by(Transaction.date.asc())
            .all()
        )

        # Get unique fund IDs from transactions
        fund_ids = set()
        portfolio_fund_ids = set()
        for t in transactions:
            fund_ids.add(t.portfolio_fund.fund_id)
            portfolio_fund_ids.add(t.portfolio_fund_id)

        # Load all fund prices for these funds up to end_date
        prices = (
            FundPrice.query.filter(
                FundPrice.fund_id.in_(fund_ids),
                FundPrice.date <= end_date,
            )
            .order_by(FundPrice.fund_id, FundPrice.date.asc())
            .all()
        )

        # Load all realized gains for these portfolios up to end_date
        realized_gains = (
            RealizedGainLoss.query.filter(
                RealizedGainLoss.portfolio_id.in_(portfolio_ids),
                RealizedGainLoss.transaction_date <= end_date,
            )
            .order_by(RealizedGainLoss.transaction_date.asc())
            .all()
        )

        # Load all dividends for these portfolio funds
        dividends = []
        if portfolio_fund_ids:
            dividends = (
                Dividend.query.filter(Dividend.portfolio_fund_id.in_(portfolio_fund_ids))
                .join(
                    Transaction,
                    Dividend.reinvestment_transaction_id == Transaction.id,
                    isouter=True,
                )
                .filter(Dividend.ex_dividend_date <= end_date)
                .with_entities(
                    Dividend.portfolio_fund_id,
                    Dividend.ex_dividend_date,
                    Transaction.shares,
                )
                .all()
            )

        return {
            "transactions": transactions,
            "prices": prices,
            "realized_gains": realized_gains,
            "dividends": dividends,
        }

    @staticmethod
    def _build_date_lookup_tables(batch_data):
        """
        Build efficient lookup tables from batch-loaded data.

        Organizes data into dictionaries and sorted lists for fast in-memory lookups
        without repeated database queries.

        Args:
            batch_data (dict): Dictionary from _load_historical_data_batch()

        Returns:
            dict: Dictionary containing:
                - prices_by_fund: {fund_id: [(date, price), ...]} sorted by date
                - transactions_by_portfolio_fund: {portfolio_fund_id: [Transaction, ...]}
                - realized_gains_by_portfolio: {portfolio_id: {fund_id: [RealizedGainLoss, ...]}}
                - dividends_by_portfolio_fund: {portfolio_fund_id: [(ex_date, shares), ...]}
        """
        from collections import defaultdict

        # Build price lookup: fund_id -> sorted list of (date, price)
        prices_by_fund = defaultdict(list)
        for price in batch_data["prices"]:
            prices_by_fund[price.fund_id].append((price.date, price.price))

        # Sort prices by date for each fund
        for fund_id in prices_by_fund:
            prices_by_fund[fund_id].sort(key=lambda x: x[0])

        # Build transaction lookup: portfolio_fund_id -> list of transactions
        transactions_by_portfolio_fund = defaultdict(list)
        for transaction in batch_data["transactions"]:
            transactions_by_portfolio_fund[transaction.portfolio_fund_id].append(transaction)

        # Build realized gains lookup: portfolio_id -> fund_id -> list of gains
        realized_gains_by_portfolio = defaultdict(lambda: defaultdict(list))
        for gain in batch_data["realized_gains"]:
            realized_gains_by_portfolio[gain.portfolio_id][gain.fund_id].append(gain)

        # Build dividend lookup: portfolio_fund_id -> list of (ex_date, shares)
        dividends_by_portfolio_fund = defaultdict(list)
        for dividend in batch_data["dividends"]:
            dividends_by_portfolio_fund[dividend.portfolio_fund_id].append(
                (dividend.ex_dividend_date, dividend.shares or 0)
            )

        # Sort dividends by date for each portfolio fund
        for pf_id in dividends_by_portfolio_fund:
            dividends_by_portfolio_fund[pf_id].sort(key=lambda x: x[0])

        return {
            "prices_by_fund": dict(prices_by_fund),
            "transactions_by_portfolio_fund": dict(transactions_by_portfolio_fund),
            "realized_gains_by_portfolio": dict(realized_gains_by_portfolio),
            "dividends_by_portfolio_fund": dict(dividends_by_portfolio_fund),
        }

    @staticmethod
    def _get_price_for_date_from_lookup(fund_id, target_date, prices_by_fund):
        """
        Get the latest price on or before a target date from lookup tables.

        Args:
            fund_id: Fund identifier
            target_date (date): Target date
            prices_by_fund (dict): Price lookup table from _build_date_lookup_tables

        Returns:
            float: Price value, or 0 if no price found
        """
        if fund_id not in prices_by_fund:
            return 0

        prices = prices_by_fund[fund_id]
        # Binary search for the latest price on or before target_date
        # Prices are sorted by date in ascending order
        result_price = 0
        for price_date, price_value in prices:
            if price_date <= target_date:
                result_price = price_value
            else:
                break

        return result_price

    @staticmethod
    def _get_dividend_shares_from_lookup(
        portfolio_fund_id, target_date, dividends_by_portfolio_fund
    ):
        """
        Get total dividend shares up to a target date from lookup tables.

        Args:
            portfolio_fund_id: Portfolio fund identifier
            target_date (date): Target date
            dividends_by_portfolio_fund (dict): Dividend lookup table

        Returns:
            float: Total dividend shares
        """
        if portfolio_fund_id not in dividends_by_portfolio_fund:
            return 0

        dividends = dividends_by_portfolio_fund[portfolio_fund_id]
        total_shares = sum(shares for ex_date, shares in dividends if ex_date <= target_date)
        return total_shares

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

        Uses eager loading and batch queries to eliminate N+1 query patterns.

        Returns:
            list: List of portfolio summary dictionaries
        """
        # Eager load portfolios with their funds and fund details
        portfolios = (
            Portfolio.query.filter_by(is_archived=False, exclude_from_overview=False)
            .options(
                selectinload(Portfolio.funds).joinedload(PortfolioFund.fund),
            )
            .all()
        )

        if not portfolios:
            return []

        # Get all portfolio and fund IDs for batch loading
        portfolio_ids = [p.id for p in portfolios]
        portfolio_fund_ids = []
        fund_ids = set()

        for portfolio in portfolios:
            for pf in portfolio.funds:
                portfolio_fund_ids.append(pf.id)
                fund_ids.add(pf.fund_id)

        # Batch load all transactions
        all_transactions = (
            Transaction.query.filter(Transaction.portfolio_fund_id.in_(portfolio_fund_ids))
            .order_by(Transaction.date.asc())
            .all()
        )

        # Group transactions by portfolio_fund_id
        transactions_by_pf = defaultdict(list)
        for t in all_transactions:
            transactions_by_pf[t.portfolio_fund_id].append(t)

        # Batch load all fund prices (latest for each fund)
        latest_prices = {}
        if fund_ids:
            price_subquery = (
                FundPrice.query.filter(FundPrice.fund_id.in_(fund_ids))
                .order_by(FundPrice.fund_id, FundPrice.date.desc())
                .all()
            )
            # Get latest price for each fund
            for price in price_subquery:
                if price.fund_id not in latest_prices:
                    latest_prices[price.fund_id] = price.price

        # Batch load all realized gains
        all_realized_gains = RealizedGainLoss.query.filter(
            RealizedGainLoss.portfolio_id.in_(portfolio_ids)
        ).all()

        # Group realized gains by portfolio_id and fund_id
        realized_gains_by_portfolio_fund = defaultdict(list)
        for gain in all_realized_gains:
            realized_gains_by_portfolio_fund[(gain.portfolio_id, gain.fund_id)].append(gain)

        # Batch load all dividends
        all_dividends_data = Dividend.query.filter(
            Dividend.portfolio_fund_id.in_(portfolio_fund_ids)
        ).all()

        # Group dividends by portfolio_fund_id
        dividends_by_pf = defaultdict(list)
        dividend_shares_by_pf = defaultdict(float)

        for dividend in all_dividends_data:
            dividends_by_pf[dividend.portfolio_fund_id].append(dividend)
            # Get dividend shares from reinvestment transaction
            if dividend.reinvestment_transaction_id:
                for t in transactions_by_pf[dividend.portfolio_fund_id]:
                    if t.id == dividend.reinvestment_transaction_id:
                        dividend_shares_by_pf[dividend.portfolio_fund_id] += t.shares or 0

        # Calculate metrics for each portfolio
        summary = []
        for portfolio in portfolios:
            portfolio_funds_data = []

            for pf in portfolio.funds:
                # Get transactions for this fund
                transactions = transactions_by_pf.get(pf.id, [])

                # Calculate shares and cost using existing method
                dividend_shares = dividend_shares_by_pf.get(pf.id, 0)
                shares, cost = PortfolioService._process_transactions_for_date(
                    transactions, datetime.now().date(), dividend_shares
                )

                # Get price
                price_value = latest_prices.get(pf.fund_id, 0)

                # Calculate current value
                current_value = shares * price_value

                # Get realized gains for this fund
                realized_records = realized_gains_by_portfolio_fund.get(
                    (pf.portfolio_id, pf.fund_id), []
                )
                realized_gain_loss = sum(r.realized_gain_loss for r in realized_records)

                # Get total dividends
                dividends = dividends_by_pf.get(pf.id, [])
                total_dividends = sum(d.total_amount for d in dividends)

                portfolio_funds_data.append(
                    {
                        "id": pf.id,
                        "fund_id": pf.fund_id,
                        "fund_name": pf.fund.name,
                        "total_shares": shares,
                        "latest_price": price_value,
                        "average_cost": cost / shares if shares > 0 else 0,
                        "total_cost": cost,
                        "current_value": current_value,
                        "unrealized_gain_loss": current_value - cost,
                        "realized_gain_loss": realized_gain_loss,
                        "total_gain_loss": realized_gain_loss + (current_value - cost),
                        "total_dividends": total_dividends,
                        "dividend_type": pf.fund.dividend_type.value,
                        "investment_type": pf.fund.investment_type.value,
                    }
                )

            if portfolio_funds_data:
                has_transactions = any(pf["total_shares"] > 0 for pf in portfolio_funds_data)

                if has_transactions:
                    summary.append(
                        PortfolioService._format_portfolio_summary(portfolio, portfolio_funds_data)
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
    def get_portfolio_history(start_date=None, end_date=None):
        """
        Get historical value data for all non-archived and visible portfolios.

        Uses batch processing to load all data once and calculate values in memory,
        avoiding repeated database queries.

        Args:
            start_date (str, optional): Start date in YYYY-MM-DD format
            end_date (str, optional): End date in YYYY-MM-DD format

        Returns:
            list: List of daily values containing portfolio history
        """
        from collections import defaultdict

        portfolios = Portfolio.query.filter_by(is_archived=False, exclude_from_overview=False).all()

        if not portfolios:
            return []

        portfolio_ids = [p.id for p in portfolios]

        # Determine date range
        # Get the earliest transaction for any portfolio
        earliest_transaction = (
            Transaction.query.join(PortfolioFund)
            .filter(PortfolioFund.portfolio_id.in_(portfolio_ids))
            .order_by(Transaction.date.asc())
            .first()
        )

        if not earliest_transaction:
            return []

        earliest_transaction_date = earliest_transaction.date

        # Parse provided dates
        if start_date:
            try:
                start_date_parsed = datetime.strptime(start_date, "%Y-%m-%d").date()
                start_date_to_use = max(earliest_transaction_date, start_date_parsed)
            except ValueError:
                start_date_to_use = earliest_transaction_date
        else:
            start_date_to_use = earliest_transaction_date

        if end_date:
            try:
                end_date_parsed = datetime.strptime(end_date, "%Y-%m-%d").date()
                end_date_to_use = min(datetime.now().date(), end_date_parsed)
            except ValueError:
                end_date_to_use = datetime.now().date()
        else:
            end_date_to_use = datetime.now().date()

        # Load all data in batch
        batch_data = PortfolioService._load_historical_data_batch(
            portfolio_ids, start_date_to_use, end_date_to_use
        )

        # Build lookup tables
        lookups = PortfolioService._build_date_lookup_tables(batch_data)

        # Build portfolio_fund mapping for quick access
        portfolio_fund_map = defaultdict(list)
        for portfolio in portfolios:
            for pf in portfolio.funds:
                portfolio_fund_map[portfolio.id].append(pf)

        # Determine each portfolio's start date
        portfolio_start_dates = {}
        for portfolio_id, portfolio_funds in portfolio_fund_map.items():
            min_date = None
            for pf in portfolio_funds:
                if pf.id in lookups["transactions_by_portfolio_fund"]:
                    transactions = lookups["transactions_by_portfolio_fund"][pf.id]
                    if transactions:
                        pf_min_date = min(t.date for t in transactions)
                        if min_date is None or pf_min_date < min_date:
                            min_date = pf_min_date
            if min_date:
                portfolio_start_dates[portfolio_id] = min_date

        # Generate history by iterating through dates
        history = []
        current_date = start_date_to_use

        while current_date <= end_date_to_use:
            daily_values = {"date": current_date.isoformat(), "portfolios": []}

            for portfolio in portfolios:
                # Skip if portfolio hasn't started yet
                if portfolio.id not in portfolio_start_dates:
                    continue
                if current_date < portfolio_start_dates[portfolio.id]:
                    continue

                total_value = 0
                total_cost = 0
                total_realized_gain = 0

                # Calculate realized gains up to this date
                if portfolio.id in lookups["realized_gains_by_portfolio"]:
                    portfolio_gains = lookups["realized_gains_by_portfolio"][portfolio.id]
                    for _fund_id, gains in portfolio_gains.items():
                        for gain in gains:
                            if gain.transaction_date <= current_date:
                                total_realized_gain += gain.realized_gain_loss

                # Calculate values for each fund
                for pf in portfolio_fund_map[portfolio.id]:
                    # Get dividend shares up to this date
                    dividend_shares = PortfolioService._get_dividend_shares_from_lookup(
                        pf.id, current_date, lookups["dividends_by_portfolio_fund"]
                    )

                    # Get transactions up to this date
                    transactions = []
                    if pf.id in lookups["transactions_by_portfolio_fund"]:
                        transactions = [
                            t
                            for t in lookups["transactions_by_portfolio_fund"][pf.id]
                            if t.date <= current_date
                        ]

                    # Calculate shares and cost
                    shares, cost = PortfolioService._process_transactions_for_date(
                        transactions, current_date, dividend_shares
                    )

                    # Get price for this date
                    price = PortfolioService._get_price_for_date_from_lookup(
                        pf.fund_id, current_date, lookups["prices_by_fund"]
                    )

                    # Calculate value
                    value = shares * price
                    total_value += value
                    total_cost += cost

                # Calculate unrealized gains
                total_unrealized_gain = total_value - total_cost

                if total_value > 0 or total_cost > 0 or total_realized_gain != 0:
                    daily_values["portfolios"].append(
                        {
                            "id": portfolio.id,
                            "name": portfolio.name,
                            "value": total_value,
                            "cost": total_cost,
                            "realized_gain": total_realized_gain,
                            "unrealized_gain": total_unrealized_gain,
                        }
                    )

            history.append(daily_values)
            current_date += timedelta(days=1)

        return history

    @staticmethod
    def get_portfolio_fund_history(portfolio_id, start_date=None, end_date=None):
        """
        Get historical value data for funds in a portfolio.

        Uses batch processing to load all data once and calculate values in memory,
        avoiding repeated database queries.

        Args:
            portfolio_id (str): Portfolio identifier
            start_date (str, optional): Start date in YYYY-MM-DD format
            end_date (str, optional): End date in YYYY-MM-DD format

        Returns:
            list: List of daily fund values
        """
        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            from flask import abort

            abort(404)

        # Get the earliest transaction date for this portfolio
        earliest_transaction = (
            Transaction.query.join(PortfolioFund)
            .filter(PortfolioFund.portfolio_id == portfolio_id)
            .order_by(Transaction.date.asc())
            .first()
        )

        if not earliest_transaction:
            return []

        earliest_date = earliest_transaction.date

        # Parse provided dates
        if start_date:
            try:
                start_date_parsed = datetime.strptime(start_date, "%Y-%m-%d").date()
                start_date_to_use = max(earliest_date, start_date_parsed)
            except ValueError:
                start_date_to_use = earliest_date
        else:
            start_date_to_use = earliest_date

        if end_date:
            try:
                end_date_parsed = datetime.strptime(end_date, "%Y-%m-%d").date()
                end_date_to_use = min(datetime.now().date(), end_date_parsed)
            except ValueError:
                end_date_to_use = datetime.now().date()
        else:
            end_date_to_use = datetime.now().date()

        # Load all data in batch for this portfolio
        batch_data = PortfolioService._load_historical_data_batch(
            [portfolio_id], start_date_to_use, end_date_to_use
        )

        # Build lookup tables
        lookups = PortfolioService._build_date_lookup_tables(batch_data)

        # Generate history by iterating through dates
        history = []
        current_date = start_date_to_use

        while current_date <= end_date_to_use:
            daily_values = {"date": current_date.isoformat(), "funds": []}

            for pf in portfolio.funds:
                # Get dividend shares up to this date
                dividend_shares = PortfolioService._get_dividend_shares_from_lookup(
                    pf.id, current_date, lookups["dividends_by_portfolio_fund"]
                )

                # Get transactions up to this date
                transactions = []
                if pf.id in lookups["transactions_by_portfolio_fund"]:
                    transactions = [
                        t
                        for t in lookups["transactions_by_portfolio_fund"][pf.id]
                        if t.date <= current_date
                    ]

                # Calculate shares and cost
                shares, cost = PortfolioService._process_transactions_for_date(
                    transactions, current_date, dividend_shares
                )

                if shares > 0:
                    # Get price for this date
                    price = PortfolioService._get_price_for_date_from_lookup(
                        pf.fund_id, current_date, lookups["prices_by_fund"]
                    )

                    # Calculate value
                    value = shares * price

                    # Calculate realized gains for this fund up to this date
                    fund_realized_gain = 0
                    if (
                        portfolio_id in lookups["realized_gains_by_portfolio"]
                        and pf.fund_id in lookups["realized_gains_by_portfolio"][portfolio_id]
                    ):
                        gains = lookups["realized_gains_by_portfolio"][portfolio_id][pf.fund_id]
                        for gain in gains:
                            if gain.transaction_date <= current_date:
                                fund_realized_gain += gain.realized_gain_loss

                    daily_values["funds"].append(
                        {
                            "portfolio_fund_id": pf.id,
                            "fund_id": pf.fund_id,
                            "fund_name": pf.fund.name,
                            "value": value,
                            "cost": cost,
                            "shares": shares,
                            "price": price,
                            "realized_gain": fund_realized_gain,
                            "unrealized_gain": value - cost,
                        }
                    )

            if daily_values["funds"]:
                history.append(daily_values)

            current_date += timedelta(days=1)

        return history

    @staticmethod
    def create_portfolio(name, description=""):
        """
        Create a new portfolio.

        Args:
            name (str): Portfolio name
            description (str, optional): Portfolio description

        Returns:
            Portfolio: Created portfolio object
        """
        from ..models import Portfolio, db

        portfolio = Portfolio(name=name, description=description)
        db.session.add(portfolio)
        db.session.commit()
        return portfolio

    @staticmethod
    def update_portfolio(portfolio_id, name, description="", exclude_from_overview=False):
        """
        Update an existing portfolio.

        Args:
            portfolio_id (str): Portfolio identifier
            name (str): Portfolio name
            description (str, optional): Portfolio description
            exclude_from_overview (bool, optional): Exclude from overview flag

        Returns:
            Portfolio: Updated portfolio object

        Raises:
            ValueError: If portfolio not found
        """
        from ..models import Portfolio, db

        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError(f"Portfolio {portfolio_id} not found")

        portfolio.name = name
        portfolio.description = description
        portfolio.exclude_from_overview = exclude_from_overview

        db.session.commit()
        return portfolio

    @staticmethod
    def delete_portfolio(portfolio_id):
        """
        Delete a portfolio.

        Args:
            portfolio_id (str): Portfolio identifier

        Returns:
            bool: True if deletion successful

        Raises:
            ValueError: If portfolio not found
        """
        from ..models import Portfolio, db

        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError(f"Portfolio {portfolio_id} not found")

        db.session.delete(portfolio)
        db.session.commit()
        return True

    @staticmethod
    def update_archive_status(portfolio_id, is_archived):
        """
        Update portfolio archive status.

        Args:
            portfolio_id (str): Portfolio identifier
            is_archived (bool): Archive status

        Returns:
            Portfolio: Updated portfolio object

        Raises:
            ValueError: If portfolio not found
        """
        from ..models import Portfolio, db

        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError(f"Portfolio {portfolio_id} not found")

        portfolio.is_archived = is_archived
        db.session.commit()
        return portfolio

    @staticmethod
    def get_portfolios_list(include_excluded=False):
        """
        Get list of portfolios with optional filtering.

        Args:
            include_excluded (bool, optional): Include portfolios excluded from overview

        Returns:
            list[Portfolio]: List of portfolio objects
        """
        from ..models import Portfolio

        query = Portfolio.query
        if not include_excluded:
            query = query.filter_by(exclude_from_overview=False)

        return query.all()

    @staticmethod
    def create_portfolio_fund(portfolio_id, fund_id):
        """
        Create a portfolio-fund relationship.

        Args:
            portfolio_id (str): Portfolio identifier
            fund_id (str): Fund identifier

        Returns:
            PortfolioFund: Created relationship object

        Raises:
            ValueError: If portfolio or fund not found
        """
        from ..models import Fund, Portfolio, PortfolioFund, db

        # Verify portfolio and fund exist
        portfolio = db.session.get(Portfolio, portfolio_id)
        if not portfolio:
            raise ValueError(f"Portfolio {portfolio_id} not found")

        fund = db.session.get(Fund, fund_id)
        if not fund:
            raise ValueError(f"Fund {fund_id} not found")

        portfolio_fund = PortfolioFund(portfolio_id=portfolio_id, fund_id=fund_id)
        db.session.add(portfolio_fund)
        db.session.commit()
        return portfolio_fund

    @staticmethod
    def delete_portfolio_fund(portfolio_fund_id, confirmed=False):
        """
        Delete a portfolio-fund relationship with cascade.

        Args:
            portfolio_fund_id (str): Portfolio-Fund relationship identifier
            confirmed (bool, optional): Confirmation for deletion with transactions

        Returns:
            dict: Deletion result with counts and names

        Raises:
            ValueError: If portfolio-fund not found or confirmation required
        """
        from ..models import Dividend, PortfolioFund, Transaction, db

        # Eager load the fund and portfolio relationships
        portfolio_fund = (
            PortfolioFund.query.options(
                db.joinedload(PortfolioFund.fund), db.joinedload(PortfolioFund.portfolio)
            )
            .filter_by(id=portfolio_fund_id)
            .first()
        )

        if not portfolio_fund:
            raise ValueError(f"Portfolio-fund relationship {portfolio_fund_id} not found")

        # Count associated transactions and dividends
        transaction_count = Transaction.query.filter_by(portfolio_fund_id=portfolio_fund_id).count()
        dividend_count = Dividend.query.filter_by(portfolio_fund_id=portfolio_fund_id).count()

        # Store fund and portfolio names before potential deletion
        fund_name = portfolio_fund.fund.name
        portfolio_name = portfolio_fund.portfolio.name

        # If there are associated records and no confirmation, raise error with details
        if (transaction_count > 0 or dividend_count > 0) and not confirmed:
            raise ValueError(
                f"Confirmation required: {transaction_count} transactions and "
                f"{dividend_count} dividends will be deleted"
            )

        # Delete associated records if they exist
        if transaction_count > 0:
            Transaction.query.filter_by(portfolio_fund_id=portfolio_fund_id).delete()
        if dividend_count > 0:
            Dividend.query.filter_by(portfolio_fund_id=portfolio_fund_id).delete()

        # Delete the portfolio-fund relationship
        db.session.delete(portfolio_fund)
        db.session.commit()

        return {
            "transactions_deleted": transaction_count,
            "dividends_deleted": dividend_count,
            "fund_name": fund_name,
            "portfolio_name": portfolio_name,
        }
