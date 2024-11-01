from flask import Blueprint, jsonify, request
from ..services.dividend_service import DividendService
from ..models import Fund, DividendType, Dividend, Transaction, db, ReinvestmentStatus
from datetime import datetime
import uuid

dividends = Blueprint('dividends', __name__)

@dividends.route('/dividends', methods=['POST'])
def create_dividend():
    data = request.json
    dividend = DividendService.create_dividend(data)
    return jsonify(DividendService.format_dividend(dividend))

@dividends.route('/dividends/fund/<string:fund_id>', methods=['GET'])
def get_fund_dividends(fund_id):
    dividends = DividendService.get_fund_dividends(fund_id)
    return jsonify([DividendService.format_dividend(d) for d in dividends])

@dividends.route('/dividends/portfolio/<string:portfolio_id>', methods=['GET'])
def get_portfolio_dividends(portfolio_id):
    dividends = DividendService.get_portfolio_dividends(portfolio_id)
    return jsonify([DividendService.format_dividend(d) for d in dividends])

@dividends.route('/funds/<string:fund_id>/dividend-type', methods=['PUT'])
def update_fund_dividend_type(fund_id):
    data = request.json
    fund = Fund.query.get_or_404(fund_id)
    fund.dividend_type = DividendType(data['dividend_type'])
    db.session.commit()
    return jsonify({
        'id': fund.id,
        'name': fund.name,
        'dividend_type': fund.dividend_type.value
    })

@dividends.route('/dividends/<string:dividend_id>', methods=['PUT'])
def update_dividend(dividend_id):
    data = request.json
    dividend = Dividend.query.get_or_404(dividend_id)
    
    try:
        # Update basic dividend information
        dividend.record_date = datetime.strptime(data['record_date'], '%Y-%m-%d').date()
        dividend.ex_dividend_date = datetime.strptime(data['ex_dividend_date'], '%Y-%m-%d').date()
        dividend.dividend_per_share = float(data['dividend_per_share'])
        
        # Update buy_order_date if provided
        if 'buy_order_date' in data and data['buy_order_date']:
            dividend.buy_order_date = datetime.strptime(data['buy_order_date'], '%Y-%m-%d').date()
        
        # Recalculate total amount
        dividend.total_amount = dividend.shares_owned * dividend.dividend_per_share
        
        # Handle stock dividend updates
        if dividend.fund.dividend_type == DividendType.STOCK:
            if data.get('reinvestment_shares') and data.get('reinvestment_price'):
                if dividend.reinvestment_transaction_id:
                    # Update existing reinvestment transaction
                    transaction = Transaction.query.get(dividend.reinvestment_transaction_id)
                    if transaction:
                        transaction.date = dividend.buy_order_date or dividend.ex_dividend_date
                        transaction.shares = float(data['reinvestment_shares'])
                        transaction.cost_per_share = float(data['reinvestment_price'])
                        db.session.add(transaction)
                        dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
                else:
                    # Create new reinvestment transaction
                    transaction = Transaction(
                        id=str(uuid.uuid4()),
                        portfolio_fund_id=dividend.portfolio_fund_id,
                        date=dividend.buy_order_date or dividend.ex_dividend_date,
                        type='dividend',
                        shares=float(data['reinvestment_shares']),
                        cost_per_share=float(data['reinvestment_price'])
                    )
                    db.session.add(transaction)
                    db.session.flush()
                    dividend.reinvestment_transaction_id = transaction.id
                    dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
            elif dividend.reinvestment_transaction_id:
                # If reinvestment data is removed, delete the transaction
                transaction = Transaction.query.get(dividend.reinvestment_transaction_id)
                if transaction:
                    db.session.delete(transaction)
                dividend.reinvestment_transaction_id = None
                dividend.reinvestment_status = ReinvestmentStatus.PENDING
        else:
            # For cash dividends, always set status to COMPLETED
            dividend.reinvestment_status = ReinvestmentStatus.COMPLETED
        
        db.session.commit()
        return jsonify(DividendService.format_dividend(dividend))
    except Exception as e:
        db.session.rollback()
        return jsonify({'error': str(e)}), 400

@dividends.route('/dividends/<string:dividend_id>', methods=['DELETE'])
def delete_dividend(dividend_id):
    try:
        DividendService.delete_dividend(dividend_id)
        return '', 204
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except Exception as e:
        return jsonify({'error': f'Unexpected error: {str(e)}'}), 500