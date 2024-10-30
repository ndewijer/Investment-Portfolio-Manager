from flask import Blueprint, jsonify, request
from ..models import Fund, PortfolioFund, db
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
        portfolios = [{'id': pf.portfolio.id, 'name': pf.portfolio.name} 
                     for pf in portfolio_funds]
        return jsonify({
            'in_use': True,
            'portfolios': portfolios
        })
    return jsonify({'in_use': False})

@funds.route('/funds/<int:fund_id>', methods=['DELETE'])
def delete_fund(fund_id):
    fund = Fund.query.get_or_404(fund_id)
    
    # Check if fund is in use
    portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
    if portfolio_funds:
        portfolios = [pf.portfolio.name for pf in portfolio_funds]
        return jsonify({
            'error': 'Cannot delete fund that is in use',
            'portfolios': portfolios
        }), 400
    
    try:
        db.session.delete(fund)
        db.session.commit()
        return '', 204
    except IntegrityError as e:
        db.session.rollback()
        return jsonify({'error': 'Database error while deleting fund'}), 500