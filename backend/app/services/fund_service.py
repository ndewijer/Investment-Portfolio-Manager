from ..models import Fund, FundPrice, db

class FundService:
    @staticmethod
    def get_all_funds():
        return Fund.query.all()
    
    @staticmethod
    def get_fund(fund_id):
        return Fund.query.get_or_404(fund_id)
    
    @staticmethod
    def create_fund(data):
        fund = Fund(
            name=data['name'],
            isin=data['isin'],
            currency=data['currency'],
            exchange=data['exchange']
        )
        db.session.add(fund)
        db.session.commit()
        return fund
    
    @staticmethod
    def update_fund(fund_id, data):
        fund = Fund.query.get_or_404(fund_id)
        fund.name = data['name']
        fund.isin = data['isin']
        fund.currency = data['currency']
        fund.exchange = data['exchange']
        db.session.commit()
        return fund
    
    @staticmethod
    def delete_fund(fund_id):
        fund = Fund.query.get_or_404(fund_id)
        db.session.delete(fund)
        db.session.commit() 