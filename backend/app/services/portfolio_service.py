from datetime import datetime, timedelta
from ..models import Portfolio, PortfolioFund, Transaction, FundPrice, Dividend, db

class PortfolioService:
    @staticmethod
    def calculate_portfolio_fund_values(portfolio_funds):
        """Calculate values for a list of portfolio funds"""
        result = []
        
        for pf in portfolio_funds:
            shares = 0
            cost = 0
            total_dividends = 0
            
            # Get all transactions sorted by date
            transactions = Transaction.query.filter_by(portfolio_fund_id=pf.id)\
                .order_by(Transaction.date.asc())\
                .all()
            
            for transaction in transactions:
                if transaction.type == 'buy':
                    # For buy transactions, exclude those from stock dividends
                    # Check if this transaction is linked to a dividend
                    dividend = Dividend.query.filter_by(reinvestment_transaction_id=transaction.id).first()
                    if not dividend:  # Only count cost if not a dividend reinvestment
                        shares += transaction.shares
                        cost += transaction.shares * transaction.cost_per_share
                    else:  # For dividend reinvestment, add shares but not cost
                        shares += transaction.shares
                elif transaction.type == 'sell':
                    # For sell transactions, subtract shares and adjust cost proportionally
                    shares -= transaction.shares
                    if shares > 0:
                        # Adjust cost proportionally to remaining shares
                        cost = (cost / (shares + transaction.shares)) * shares
                    else:
                        cost = 0
            
            # Calculate total dividends paid
            dividends = Dividend.query.filter_by(portfolio_fund_id=pf.id).all()
            for dividend in dividends:
                total_dividends += dividend.total_amount
            
            # Get latest price
            latest_price = FundPrice.query.filter_by(fund_id=pf.fund_id)\
                .order_by(FundPrice.date.desc())\
                .first()
            
            current_value = shares * (latest_price.price if latest_price else 0)
            
            result.append({
                'id': pf.id,
                'fund_id': pf.fund_id,
                'fund_name': pf.fund.name,
                'total_shares': shares,
                'latest_price': latest_price.price if latest_price else 0,
                'average_cost': cost / shares if shares > 0 else 0,
                'total_cost': cost,
                'current_value': current_value,
                'total_dividends': total_dividends,
                'dividend_type': pf.fund.dividend_type.value
            })
        
        return result

    @staticmethod
    def get_portfolio_summary():
        portfolios = Portfolio.query.filter_by(is_archived=False).all()
        summary = []
        
        for portfolio in portfolios:
            has_transactions = False
            portfolio_funds_data = PortfolioService.calculate_portfolio_fund_values(portfolio.funds)
            
            if portfolio_funds_data:
                total_value = sum(pf['current_value'] for pf in portfolio_funds_data)
                total_cost = sum(pf['total_cost'] for pf in portfolio_funds_data)
                total_dividends = sum(pf['total_dividends'] for pf in portfolio_funds_data)
                has_transactions = any(pf['total_shares'] > 0 for pf in portfolio_funds_data)
                
                if has_transactions:
                    summary.append({
                        'id': portfolio.id,
                        'name': portfolio.name,
                        'totalValue': total_value,
                        'totalCost': total_cost,
                        'totalDividends': total_dividends,
                        'is_archived': portfolio.is_archived
                    })
        
        return summary

    @staticmethod
    def get_portfolio_funds(portfolio_id):
        portfolio_funds = PortfolioFund.query.filter_by(portfolio_id=portfolio_id).all()
        return PortfolioService.calculate_portfolio_fund_values(portfolio_funds)

    @staticmethod
    def get_all_portfolio_funds():
        portfolio_funds = PortfolioFund.query.all()
        return [{
            'id': pf.id,
            'portfolio_id': pf.portfolio_id,
            'fund_id': pf.fund_id,
            'portfolio_name': pf.portfolio.name,
            'fund_name': pf.fund.name
        } for pf in portfolio_funds]

    @staticmethod
    def get_portfolio_history():
        portfolios = Portfolio.query.filter_by(is_archived=False).all()
        history = []
        
        portfolio_date_ranges = {}
        for portfolio in portfolios:
            portfolio_transactions = Transaction.query.join(
                PortfolioFund
            ).filter(
                PortfolioFund.portfolio_id == portfolio.id
            ).order_by(Transaction.date).all()
            
            if portfolio_transactions:
                portfolio_date_ranges[portfolio.id] = {
                    'start_date': portfolio_transactions[0].date,
                    'end_date': datetime.now().date()
                }
        
        if not portfolio_date_ranges:
            return []
        
        one_year_ago = datetime.now().date() - timedelta(days=365)
        start_date = max(one_year_ago, min(range['start_date'] for range in portfolio_date_ranges.values()))
        current_date = start_date
        today = datetime.now().date()
        
        while current_date <= today:
            daily_values = PortfolioService._calculate_daily_values(
                current_date, portfolios, portfolio_date_ranges)
            history.append(daily_values)
            current_date += timedelta(days=1)
        
        return history

    @staticmethod
    def _calculate_daily_values(current_date, portfolios, portfolio_date_ranges):
        daily_values = {
            'date': current_date.isoformat(),
            'portfolios': []
        }
        
        for portfolio in portfolios:
            if (portfolio.id not in portfolio_date_ranges or 
                current_date < portfolio_date_ranges[portfolio.id]['start_date']):
                continue
                
            portfolio_value = PortfolioService._calculate_portfolio_value(
                portfolio, current_date)
            
            if portfolio_value['total_value'] > 0 or portfolio_value['total_cost'] > 0:
                daily_values['portfolios'].append({
                    'id': portfolio.id,
                    'name': portfolio.name,
                    'value': portfolio_value['total_value'],
                    'cost': portfolio_value['total_cost']
                })
        
        return daily_values

    @staticmethod
    def _calculate_portfolio_value(portfolio, date):
        total_value = 0
        total_cost = 0
        
        for pf in portfolio.funds:
            shares = 0
            cost = 0
            
            for transaction in pf.transactions:
                if transaction.date <= date:
                    if transaction.type == 'buy':
                        shares += transaction.shares
                        cost += transaction.shares * transaction.cost_per_share
                    else:
                        shares -= transaction.shares
                        cost = (cost / (shares + transaction.shares)) * shares if shares > 0 else 0
            
            price = FundPrice.query.filter(
                FundPrice.fund_id == pf.fund_id,
                FundPrice.date <= date
            ).order_by(FundPrice.date.desc()).first()
            
            if price and shares > 0:
                total_value += shares * price.price
                total_cost += cost
        
        return {
            'total_value': total_value,
            'total_cost': total_cost
        }

    @staticmethod
    def get_portfolio_fund_history(portfolio_id):
        portfolio = Portfolio.query.get_or_404(portfolio_id)
        history = []
        
        # Get the earliest transaction date for this portfolio
        earliest_transaction = Transaction.query.join(
            PortfolioFund
        ).filter(
            PortfolioFund.portfolio_id == portfolio_id
        ).order_by(Transaction.date).first()
        
        if not earliest_transaction:
            return []
        
        start_date = earliest_transaction.date
        current_date = start_date
        today = datetime.now().date()
        
        while current_date <= today:
            daily_values = {
                'date': current_date.isoformat(),
                'funds': []
            }
            
            for pf in portfolio.funds:
                shares = 0
                cost = 0
                
                # Calculate shares and cost up to current_date
                for transaction in pf.transactions:
                    if transaction.date <= current_date:
                        if transaction.type == 'buy':
                            shares += transaction.shares
                            cost += transaction.shares * transaction.cost_per_share
                        else:
                            shares -= transaction.shares
                            cost = (cost / (shares + transaction.shares)) * shares if shares > 0 else 0
                
                # Get the price for this date
                price = FundPrice.query.filter(
                    FundPrice.fund_id == pf.fund_id,
                    FundPrice.date <= current_date
                ).order_by(FundPrice.date.desc()).first()
                
                if price and shares > 0:
                    value = shares * price.price
                    daily_values['funds'].append({
                        'fund_id': pf.fund.id,
                        'fund_name': pf.fund.name,
                        'value': value,
                        'cost': cost,
                        'shares': shares,
                        'price': price.price
                    })
            
            if daily_values['funds']:
                history.append(daily_values)
            current_date += timedelta(days=1)
        
        return history