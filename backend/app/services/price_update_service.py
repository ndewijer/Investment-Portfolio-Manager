"""
Service class for updating fund prices.

This module provides methods for:
- Updating today's prices
- Updating historical prices
"""

from datetime import UTC, datetime, timedelta

import yfinance as yf

from ..models import Fund, FundPrice, PortfolioFund, Transaction, db
from ..services.logging_service import LogCategory, LogLevel, logger


class TodayPriceService:
    """
    Service class for updating today's fund prices.

    Provides methods for fetching and storing the latest available
    fund prices from external sources.
    """

    @staticmethod
    def get_latest_available_date():
        """
        Get yesterday's date as the latest available date.

        Returns:
            date: Yesterday's date in UTC
        """
        return datetime.now(UTC).date() - timedelta(days=1)

    @staticmethod
    def update_todays_price(fund_id):
        """
        Update the latest available price (yesterday) for a fund.

        Args:
            fund_id (str): Fund identifier

        Returns:
            tuple: (response dict, status code) containing:
                - response: Log message and details
                - status: HTTP status code

        Raises:
            Exception: If price update fails
        """
        try:
            fund = db.session.get(Fund, fund_id)
            if not fund.symbol:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"No symbol available for fund {fund.name}",
                    details={"fund_id": fund_id},
                    http_status=400,
                )
                return response, status

            latest_date = TodayPriceService.get_latest_available_date()

            # Check if yesterday's price already exists - use exact date comparison
            existing_price = FundPrice.query.filter_by(fund_id=fund_id, date=latest_date).first()

            if existing_price:
                response, status = logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message="Latest price already exists",
                    details={
                        "fund_id": fund_id,
                        "date": latest_date.isoformat(),
                        "price": existing_price.price,
                    },
                    http_status=200,
                )
                return response, status

            # Fetch latest price (yesterday's)
            ticker = yf.Ticker(fund.symbol)
            end_date = datetime.now(UTC).date()  # Today
            start_date = end_date - timedelta(
                days=5
            )  # Get a few days of data to ensure we get the latest
            history = ticker.history(start=start_date, end=end_date)

            if not history.empty:
                # Get the last available price
                last_date = history.index[-1].date()
                last_price = float(history["Close"].iloc[-1])

                # Upsert: Update existing price or create new one
                existing_price = FundPrice.query.filter_by(fund_id=fund_id, date=last_date).first()

                if existing_price:
                    # Update existing price
                    price_changed = existing_price.price != last_price
                    existing_price.price = last_price
                    price = existing_price
                else:
                    # Create new price record
                    price = FundPrice(fund_id=fund_id, date=last_date, price=last_price)
                    db.session.add(price)
                    price_changed = True

                db.session.commit()

                # Invalidate materialized view only if price changed
                if price_changed:
                    try:
                        from .portfolio_history_materialized_service import (
                            PortfolioHistoryMaterializedService,
                        )

                        PortfolioHistoryMaterializedService.invalidate_from_price_update(
                            fund_id, last_date
                        )
                    except Exception:
                        # Don't fail price update if invalidation fails
                        pass

                action = "Updated" if existing_price else "Added"
                response, status = logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message=f"{action} latest price for fund {fund.name}",
                    details={
                        "fund_id": fund_id,
                        "date": last_date.isoformat(),
                        "price": last_price,
                        "updated": existing_price is not None,
                    },
                    http_status=200,
                )
                return response, status
            else:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message="No recent price data available",
                    details={
                        "fund_id": fund_id,
                        "start_date": start_date.isoformat(),
                        "end_date": end_date.isoformat(),
                    },
                    http_status=404,
                )
                return response, status

        except Exception as e:
            db.session.rollback()
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating latest price: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
                http_status=500,
            )
            return response, status


class HistoricalPriceService:
    """
    Service class for managing historical fund prices.

    Provides methods for:
    - Finding missing price dates
    - Updating historical price data
    - Managing price history
    """

    @staticmethod
    def get_oldest_transaction_date(fund_id):
        """
        Get the date of the oldest transaction for a fund.

        Args:
            fund_id (str): Fund identifier

        Returns:
            date: Date of oldest transaction or None if no transactions
        """
        oldest_transaction = (
            Transaction.query.join(PortfolioFund)
            .filter(PortfolioFund.fund_id == fund_id)
            .order_by(Transaction.date.asc())
            .first()
        )

        return oldest_transaction.date if oldest_transaction else None

    @staticmethod
    def get_missing_dates(fund_id):
        """
        Find missing dates in price history.

        Args:
            fund_id (str): Fund identifier

        Returns:
            list: List of dates missing price data
        """
        start_date = HistoricalPriceService.get_oldest_transaction_date(fund_id)
        if not start_date:
            return []

        existing_dates = {price.date for price in FundPrice.query.filter_by(fund_id=fund_id).all()}

        all_dates = []
        current_date = start_date
        today = datetime.now().date()

        while current_date <= today:
            if current_date not in existing_dates:
                all_dates.append(current_date)
            current_date += timedelta(days=1)

        return all_dates

    @staticmethod
    def update_historical_prices(fund_id):
        """
        Update historical prices for a fund.

        Args:
            fund_id (str): Fund identifier

        Returns:
            tuple: (response dict, status code) containing:
                - response: Log message and update details
                - status: HTTP status code

        Raises:
            Exception: If historical price update fails
        """
        try:
            fund = db.session.get(Fund, fund_id)
            if not fund.symbol:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"No symbol available for fund {fund.name}",
                    details={"fund_id": fund_id},
                    http_status=400,
                )
                return response, status

            missing_dates = HistoricalPriceService.get_missing_dates(fund_id)
            if not missing_dates:
                response, status = logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message="No missing dates to update",
                    details={"fund_id": fund_id},
                    http_status=200,
                )
                response["newPrices"] = False
                return response, status

            # Fetch historical data
            ticker = yf.Ticker(fund.symbol)
            start_date = min(missing_dates)
            end_date = max(missing_dates) + timedelta(days=1)
            history = ticker.history(start=start_date, end=end_date)

            # Convert history index dates to date objects for comparison
            history_dates = {pd_date.date(): pd_date for pd_date in history.index}

            # Update prices with upsert logic
            updated_count = 0
            for date in missing_dates:
                if date in history_dates:
                    pd_date = history_dates[date]  # Get the original pandas datetime
                    new_price = float(history.loc[pd_date]["Close"])

                    # Upsert: Update existing or create new
                    existing_price = FundPrice.query.filter_by(fund_id=fund_id, date=date).first()

                    if existing_price:
                        # Update existing price
                        # (shouldn't happen for "missing" dates, but safety net)
                        if existing_price.price != new_price:
                            existing_price.price = new_price
                            updated_count += 1
                    else:
                        # Create new price record
                        price = FundPrice(fund_id=fund_id, date=date, price=new_price)
                        db.session.add(price)
                        updated_count += 1

            db.session.commit()

            # Invalidate materialized view if any prices were updated
            if updated_count > 0:
                try:
                    from .portfolio_history_materialized_service import (
                        PortfolioHistoryMaterializedService,
                    )

                    PortfolioHistoryMaterializedService.invalidate_from_price_update(
                        fund_id, start_date
                    )
                except Exception:
                    # Don't fail price update if invalidation fails
                    pass

            response, status = logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Updated historical prices for fund {fund.name}",
                details={
                    "fund_id": fund_id,
                    "updated_count": updated_count,
                    "missing_dates": len(missing_dates),
                    "date_range": f"{start_date} to {end_date}",
                },
                http_status=200,
            )
            # Add newPrices field for frontend auto-refresh
            response["newPrices"] = updated_count > 0
            return response, status

        except Exception as e:
            db.session.rollback()
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating historical prices: {e!s}",
                details={"fund_id": fund_id, "error": str(e)},
                http_status=500,
            )
            return response, status
