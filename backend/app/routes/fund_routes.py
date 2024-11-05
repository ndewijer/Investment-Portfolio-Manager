from flask import Blueprint, jsonify, request
from ..models import Fund, PortfolioFund, Transaction, db, DividendType, LogLevel, LogCategory, FundPrice
from ..services.fund_service import FundService
from ..services.symbol_lookup_service import SymbolLookupService
from ..services.logging_service import logger, track_request
from sqlalchemy.exc import IntegrityError
import yfinance as yf

funds = Blueprint('funds', __name__)

@funds.route('/funds', methods=['GET'])
@track_request
def get_funds():
    try:
        funds = Fund.query.all()
        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message="Successfully retrieved all funds",
            details={'fund_count': len(funds)}
        )
        return jsonify([{
            'id': f.id,
            'name': f.name,
            'isin': f.isin,
            'symbol': f.symbol,
            'currency': f.currency,
            'exchange': f.exchange,
            'dividend_type': f.dividend_type.value,
            'investment_type': f.investment_type.value
        } for f in funds])
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error retrieving funds: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@funds.route('/funds', methods=['POST'])
@track_request
def create_fund():
    try:
        data = request.json
        
        # If symbol is provided, try to get symbol info before creating fund
        if 'symbol' in data and data['symbol']:
            try:
                symbol_info = SymbolLookupService.get_symbol_info(data['symbol'], force_refresh=True)
                if symbol_info:
                    logger.log(
                        level=LogLevel.INFO,
                        category=LogCategory.FUND,
                        message=f"Successfully retrieved symbol info for {data['symbol']}",
                        details=symbol_info
                    )
            except Exception as e:
                logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"Failed to retrieve symbol info: {str(e)}",
                    details={'symbol': data['symbol']}
                )
        
        fund = Fund(
            name=data['name'],
            isin=data['isin'],
            symbol=data.get('symbol'),
            currency=data['currency'],
            exchange=data['exchange'],
            dividend_type=DividendType.NONE
        )
        db.session.add(fund)
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully created fund {fund.name}",
            details={
                'fund_id': fund.id,
                'isin': fund.isin,
                'symbol': fund.symbol
            }
        )

        return jsonify({
            'id': fund.id,
            'name': fund.name,
            'isin': fund.isin,
            'symbol': fund.symbol,
            'currency': fund.currency,
            'exchange': fund.exchange,
            'dividend_type': fund.dividend_type.value
        })
    except IntegrityError as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message="ISIN must be unique",
            details={
                'user_message': 'A fund with this ISIN already exists',
                'error': str(e)
            },
            http_status=400
        )
        return jsonify(response), status
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error creating fund: {str(e)}",
            details={
                'user_message': 'Error creating fund',
                'error': str(e)
            },
            http_status=500
        )
        return jsonify(response), status

@funds.route('/funds/<string:fund_id>', methods=['GET'])
def get_fund(fund_id):
    fund = Fund.query.get_or_404(fund_id)
    return jsonify({
        'id': fund.id,
        'name': fund.name,
        'isin': fund.isin,
        'symbol': fund.symbol,
        'currency': fund.currency,
        'exchange': fund.exchange,
        'dividend_type': fund.dividend_type.value,
        'investment_type': fund.investment_type.value
    })

@funds.route('/funds/<string:fund_id>', methods=['PUT'])
@track_request
def update_fund(fund_id):
    try:
        data = request.json
        fund = Fund.query.get_or_404(fund_id)
        fund.name = data['name']
        fund.isin = data['isin']
        
        # Handle symbol update
        if 'symbol' in data and data['symbol']:
            old_symbol = fund.symbol
            new_symbol = data['symbol']
            
            # Only lookup if symbol has changed
            if old_symbol != new_symbol:
                fund.symbol = new_symbol
                # Try to get symbol info and store it
                try:
                    symbol_info = SymbolLookupService.get_symbol_info(new_symbol, force_refresh=True)
                    if symbol_info:
                        logger.log(
                            level=LogLevel.INFO,
                            category=LogCategory.FUND,
                            message=f"Successfully retrieved symbol info for {new_symbol}",
                            details=symbol_info
                        )
                except Exception as e:
                    logger.log(
                        level=LogLevel.WARNING,
                        category=LogCategory.FUND,
                        message=f"Failed to retrieve symbol info: {str(e)}",
                        details={'symbol': new_symbol}
                    )
        else:
            fund.symbol = None  # Clear symbol if not provided
            
        fund.currency = data['currency']
        fund.exchange = data['exchange']
        if 'dividend_type' in data:
            fund.dividend_type = DividendType(data['dividend_type'])
            
        db.session.add(fund)
        db.session.commit()

        return jsonify({
            'id': fund.id,
            'name': fund.name,
            'isin': fund.isin,
            'symbol': fund.symbol,
            'currency': fund.currency,
            'exchange': fund.exchange,
            'dividend_type': fund.dividend_type.value
        })
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error updating fund: {str(e)}",
            details={
                'fund_id': fund_id,
                'error': str(e),
                'request_data': data
            },
            http_status=400
        )
        return jsonify(response), status

@funds.route('/funds/<string:fund_id>/check-usage', methods=['GET'])
@track_request
def check_fund_usage(fund_id):
    try:
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
                logger.log(
                    level=LogLevel.INFO,
                    category=LogCategory.FUND,
                    message=f"Fund {fund_id} is in use",
                    details={
                        'in_use': True,
                        'portfolios': portfolio_data
                    }
                )
                return jsonify({
                    'in_use': True,
                    'portfolios': portfolio_data
                })
        return jsonify({'in_use': False})
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error checking fund usage: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@funds.route('/funds/<string:fund_id>', methods=['DELETE'])
@track_request
def delete_fund(fund_id):
    try:
        fund = Fund.query.get_or_404(fund_id)
        
        # Check if fund has any transactions
        portfolio_funds = PortfolioFund.query.filter_by(fund_id=fund_id).all()
        for pf in portfolio_funds:
            transaction_count = Transaction.query.filter_by(portfolio_fund_id=pf.id).count()
            if transaction_count > 0:
                response, status = logger.log(
                    level=LogLevel.WARNING,
                    category=LogCategory.FUND,
                    message=f"Cannot delete fund {fund_id} with existing transactions",
                    details={
                        'user_message': 'Cannot delete fund that has transactions',
                        'portfolios': [{
                            'name': pf.portfolio.name,
                            'transaction_count': transaction_count
                        } for pf in portfolio_funds if Transaction.query.filter_by(portfolio_fund_id=pf.id).count() > 0]
                    },
                    http_status=400
                )
                return jsonify(response), status
        
        db.session.delete(fund)
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully deleted fund {fund_id}",
            details={'fund_name': fund.name}
        )

        return '', 204
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error deleting fund: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@funds.route('/lookup-symbol-info/<string:symbol>', methods=['GET'])
@track_request
def lookup_symbol_info(symbol):
    try:
        # Use SymbolLookupService to get info (checks cache first)
        force_refresh = request.args.get('force_refresh', 'false').lower() == 'true'
        symbol_info = SymbolLookupService.get_symbol_info(symbol, force_refresh=force_refresh)
        
        if symbol_info:
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.FUND,
                message=f"Successfully retrieved symbol info for {symbol}",
                details={
                    'symbol': symbol,
                    'source': 'cache' if not force_refresh else 'yfinance',
                    'info': symbol_info
                }
            )
            return jsonify(symbol_info)
        else:
            response, status = logger.log(
                level=LogLevel.WARNING,
                category=LogCategory.FUND,
                message=f"No information found for symbol {symbol}",
                details={'symbol': symbol},
                http_status=404
            )
            return jsonify(response), status
            
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error looking up symbol: {str(e)}",
            details={
                'symbol': symbol,
                'error': str(e)
            },
            http_status=500
        )
        return jsonify(response), status

# Add this new route to get fund prices
@funds.route('/fund-prices/<string:fund_id>', methods=['GET'])
@track_request
def get_fund_prices(fund_id):
    try:
        # Get the fund to ensure it exists
        fund = Fund.query.get_or_404(fund_id)
        
        # Get all prices for this fund, ordered by date
        prices = FundPrice.query.filter_by(fund_id=fund_id)\
            .order_by(FundPrice.date.desc())\
            .all()
        
        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.FUND,
            message=f"Successfully retrieved price history for fund {fund.name}",
            details={
                'fund_id': fund_id,
                'price_count': len(prices)
            }
        )

        return jsonify([{
            'id': price.id,
            'date': price.date.isoformat(),
            'price': price.price
        } for price in prices])
        
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.FUND,
            message=f"Error retrieving fund prices: {str(e)}",
            details={
                'fund_id': fund_id,
                'error': str(e)
            },
            http_status=500
        )
        return jsonify(response), status