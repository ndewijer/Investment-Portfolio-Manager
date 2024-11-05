from datetime import datetime, timedelta, UTC
from ..models import Fund, FundPrice, Transaction, PortfolioFund, db
from ..services.logging_service import logger, LogLevel, LogCategory
import yfinance as yf
import pandas as pd

class TodayPriceService:
    @staticmethod
    def get_latest_available_date():
        """Get yesterday's date as the latest available date"""
        return datetime.now(UTC).date() - timedelta(days=1)

    @staticmethod
    def update_todays_price(fund_id):
        """Update the latest available price (yesterday) for a fund"""
        try:
            fund = Fund.query.get(fund_id)
            if not fund.symbol:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"No symbol available for fund {fund.name}",
                    details={'fund_id': fund_id},
                    http_status=400
                )
                return response, status

            latest_date = TodayPriceService.get_latest_available_date()
            
            # Check if yesterday's price already exists - use exact date comparison
            existing_price = FundPrice.query.filter_by(
                fund_id=fund_id, 
                date=latest_date
            ).first()
            
            if existing_price:
                response, status = logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message="Latest price already exists",
                    details={
                        'fund_id': fund_id,
                        'date': latest_date.isoformat(),
                        'price': existing_price.price
                    },
                    http_status=200
                )
                return response, status

            # Fetch latest price (yesterday's)
            ticker = yf.Ticker(fund.symbol)
            end_date = datetime.now(UTC).date()  # Today
            start_date = end_date - timedelta(days=5)  # Get a few days of data to ensure we get the latest
            history = ticker.history(start=start_date, end=end_date)
            
            if not history.empty:
                # Get the last available price
                last_date = history.index[-1].date()
                
                # Only add if we don't already have this date
                if not FundPrice.query.filter_by(fund_id=fund_id, date=last_date).first():
                    last_price = float(history['Close'][-1])
                    price = FundPrice(
                        fund_id=fund_id,
                        date=last_date,
                        price=last_price
                    )
                    db.session.add(price)
                    db.session.commit()

                    response, status = logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.FUND,
                        message=f"Updated latest price for fund {fund.name}",
                        details={
                            'fund_id': fund_id,
                            'date': last_date.isoformat(),
                            'price': last_price
                        },
                        http_status=200
                    )
                    return response, status
                else:
                    response, status = logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.FUND,
                        message="Price for latest available date already exists",
                        details={
                            'fund_id': fund_id,
                            'date': last_date.isoformat()
                        },
                        http_status=200
                    )
                    return response, status
            else:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message="No recent price data available",
                    details={
                        'fund_id': fund_id,
                        'start_date': start_date.isoformat(),
                        'end_date': end_date.isoformat()
                    },
                    http_status=404
                )
                return response, status

        except Exception as e:
            db.session.rollback()
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating latest price: {str(e)}",
                details={
                    'fund_id': fund_id,
                    'error': str(e)
                },
                http_status=500
            )
            return response, status

class HistoricalPriceService:
    @staticmethod
    def get_oldest_transaction_date(fund_id):
        """Get the date of the oldest transaction for a fund"""
        oldest_transaction = Transaction.query.join(
            PortfolioFund
        ).filter(
            PortfolioFund.fund_id == fund_id
        ).order_by(
            Transaction.date.asc()
        ).first()
        
        return oldest_transaction.date if oldest_transaction else None

    @staticmethod
    def get_missing_dates(fund_id):
        """Find missing dates in price history"""
        start_date = HistoricalPriceService.get_oldest_transaction_date(fund_id)
        if not start_date:
            return []

        existing_dates = set(
            price.date for price in FundPrice.query.filter_by(fund_id=fund_id).all()
        )
        
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
        """Update historical prices for a fund"""
        try:
            fund = Fund.query.get(fund_id)
            if not fund.symbol:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"No symbol available for fund {fund.name}",
                    details={'fund_id': fund_id},
                    http_status=400
                )
                return response, status

            missing_dates = HistoricalPriceService.get_missing_dates(fund_id)
            if not missing_dates:
                response, status = logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message="No missing dates to update",
                    details={'fund_id': fund_id},
                    http_status=200
                )
                return response, status

            # Fetch historical data
            ticker = yf.Ticker(fund.symbol)
            start_date = min(missing_dates)
            end_date = max(missing_dates) + timedelta(days=1)
            history = ticker.history(start=start_date, end=end_date)

            # Convert history index dates to date objects for comparison
            history_dates = {pd_date.date(): pd_date for pd_date in history.index}

            # Update prices
            updated_count = 0
            for date in missing_dates:
                if date in history_dates:
                    pd_date = history_dates[date]  # Get the original pandas datetime
                    price = FundPrice(
                        fund_id=fund_id,
                        date=date,
                        price=float(history.loc[pd_date]['Close'])
                    )
                    db.session.add(price)
                    updated_count += 1

            db.session.commit()

            response, status = logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Updated historical prices for fund {fund.name}",
                details={
                    'fund_id': fund_id,
                    'updated_count': updated_count,
                    'missing_dates': len(missing_dates),
                    'date_range': f"{start_date} to {end_date}"
                },
                http_status=200
            )
            return response, status

        except Exception as e:
            db.session.rollback()
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.FUND,
                message=f"Error updating historical prices: {str(e)}",
                details={
                    'fund_id': fund_id,
                    'error': str(e)
                },
                http_status=500
            )
            return response, status