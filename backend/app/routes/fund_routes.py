from flask import Blueprint, jsonify, request
from ..models import Fund, PortfolioFund, Transaction, db
from ..services.fund_service import FundService
from sqlalchemy.exc import IntegrityError

funds = Blueprint('funds', __name__)

@funds.route('/funds', methods=['GET'])
def get_funds():
    funds = Fund.query.all()
    return jsonify([{
        'id': f.id,
        'name': f.name,
        'isin': f.isin,
        'currency': f.currency,
        'exchange': f.exchange
    } for f in funds])

@funds.route('/funds', methods=['POST'])
def create_fund():
    data = request.json
    fund = Fund(
        name=data['name'],
        isin=data['isin'],
        currency=data['currency'],
        exchange=data['exchange']
    )
    db.session.add(fund)
    db.session.commit()
    return jsonify({
        'id': fund.id,
        'name': fund.name,
        'isin': fund.isin,
        'currency': fund.currency,
        'exchange': fund.exchange
    })

@funds.route('/funds/<int:fund_id>', methods=['GET'])
def get_fund(fund_id):
    fund = Fund.query.get_or_404(fund_id)
    return jsonify({
        'id': fund.id,
        'name': fund.name,
        'isin': fund.isin,
        'currency': fund.currency,
        'exchange': fund.exchange
    })

@funds.route('/funds/<int:fund_id>', methods=['PUT'])
def update_fund(fund_id):
    data = request.json
    fund = Fund.query.get_or_404(fund_id)
    fund.name = data['name']
    fund.isin = data['isin']
    fund.currency = data['currency']
    fund.exchange = data['exchange']
    db.session.add(fund)
    db.session.commit()
    return jsonify({
        'id': fund.id,
        'name': fund.name,
        'isin': fund.isin,
        'currency': fund.currency,
        'exchange': fund.exchange
    })

@funds.route('/funds/<int:fund_id>/check-usage', methods=['GET'])
def check_fund_usage(fund_id):
    portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
    if portfolio_funds:
        # Get portfolios and their transaction counts
        portfolio_data = []
        for pf in portfolio_funds:
            transaction_count = Transaction.query.filter_by(portfolio_fund_id=pf.id).count()
            if transaction_count > 0:
                portfolio_data.append({
                    'id': pf.portfolio.id, 
                    'name': pf.portfolio.name,
                    'transaction_count': transaction_count
                })
        
        if portfolio_data:
            return jsonify({
                'in_use': True,
                'portfolios': portfolio_data
            })
    return jsonify({'in_use': False})

@funds.route('/funds/<int:fund_id>', methods=['DELETE'])
def delete_fund(fund_id):
    fund = Fund.query.get_or_404(fund_id)
    
    # Check if fund has any transactions
    portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
    for pf in portfolio_funds:
        transaction_count = Transaction.query.filter_by(portfolio_fund_id=pf.id).count()
        if transaction_count > 0:
            return jsonify({
                'error': 'Cannot delete fund that has transactions',
                'portfolios': [{
                    'name': pf.portfolio.name,
                    'transaction_count': transaction_count
                } for pf in portfolio_funds if Transaction.query.filter_by(portfolio_fund_id=pf.id).count() > 0]
            }), 400
    
    try:
        db.session.delete(fund)
        db.session.commit()
        return '', 204
    except IntegrityError as e:
        db.session.rollback()
        return jsonify({'error': 'Database error while deleting fund'}), 500