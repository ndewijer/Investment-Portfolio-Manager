from datetime import datetime
from ..models import Transaction, PortfolioFund, db

class TransactionService:
    @staticmethod
    def get_all_transactions():
        transactions = Transaction.query.all()
        return [TransactionService.format_transaction(t) for t in transactions]

    @staticmethod
    def get_portfolio_transactions(portfolio_id):
        transactions = Transaction.query.join(
            PortfolioFund
        ).filter(
            PortfolioFund.portfolio_id == portfolio_id
        ).all()
        return [TransactionService.format_transaction(t) for t in transactions]

    @staticmethod
    def format_transaction(transaction):
        return {
            'id': transaction.id,
            'portfolio_fund_id': transaction.portfolio_fund_id,
            'fund_name': transaction.portfolio_fund.fund.name,
            'date': transaction.date.isoformat(),
            'type': transaction.type,
            'shares': transaction.shares,
            'cost_per_share': transaction.cost_per_share
        }
    
    @staticmethod
    def create_transaction(data):
        transaction = Transaction(
            portfolio_fund_id=data['portfolio_fund_id'],
            date=datetime.strptime(data['date'], '%Y-%m-%d').date(),
            type=data['type'],
            shares=data['shares'],
            cost_per_share=data['cost_per_share']
        )
        db.session.add(transaction)
        db.session.commit()
        return transaction
    
    @staticmethod
    def update_transaction(transaction_id, data):
        transaction = Transaction.query.get_or_404(transaction_id)
        transaction.date = datetime.strptime(data['date'], '%Y-%m-%d').date()
        transaction.type = data['type']
        transaction.shares = data['shares']
        transaction.cost_per_share = data['cost_per_share']
        db.session.commit()
        return transaction
    
    @staticmethod
    def delete_transaction(transaction_id):
        transaction = Transaction.query.get_or_404(transaction_id)
        db.session.delete(transaction)
        db.session.commit() 