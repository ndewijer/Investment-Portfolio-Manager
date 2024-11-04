from flask import Blueprint, jsonify, request
from ..services.developer_service import DeveloperService
from datetime import datetime, timedelta
from ..models import (
    PortfolioFund, LogLevel, LogCategory, db, SystemSetting, 
    SystemSettingKey, Log
)
from ..services.logging_service import logger, track_request
import json

developer = Blueprint('developer', __name__)

@developer.route('/exchange-rate', methods=['GET'])
@track_request
def get_exchange_rate():
    try:
        from_currency = request.args.get('from_currency')
        to_currency = request.args.get('to_currency')
        date = request.args.get('date')
        
        if date:
            date = datetime.strptime(date, '%Y-%m-%d').date()
        else:
            date = datetime.now().date()
            
        exchange_rate = DeveloperService.get_exchange_rate(from_currency, to_currency, date)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully retrieved exchange rate",
            details={
                'from_currency': from_currency,
                'to_currency': to_currency,
                'date': date.isoformat(),
                'rate': exchange_rate['rate'] if exchange_rate else None
            }
        )

        return jsonify(exchange_rate)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving exchange rate: {str(e)}",
            details={
                'from_currency': from_currency,
                'to_currency': to_currency,
                'date': date.isoformat() if date else None,
                'error': str(e)
            },
            http_status=500
        )
        return jsonify(response), status

@developer.route('/exchange-rate', methods=['POST'])
@track_request
def set_exchange_rate():
    try:
        data = request.json
        result = DeveloperService.set_exchange_rate(
            data['from_currency'],
            data['to_currency'],
            data['rate'],
            datetime.strptime(data.get('date', None), '%Y-%m-%d').date() if data.get('date') else None
        )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully set exchange rate",
            details={
                'from_currency': result['from_currency'],
                'to_currency': result['to_currency'],
                'rate': result['rate'],
                'date': result['date']
            }
        )

        return jsonify(result)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error setting exchange rate: {str(e)}",
            details={
                'request_data': data,
                'error': str(e)
            },
            http_status=400
        )
        return jsonify(response), status

@developer.route('/import-transactions', methods=['POST'])
@track_request
def import_transactions():
    try:
        if 'file' not in request.files:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message="No file provided for transaction import",
                details={'user_message': 'No file provided'},
                http_status=400
            )
            return jsonify(response), status
            
        file = request.files['file']
        portfolio_fund_id = request.form.get('fund_id')
        
        if not portfolio_fund_id:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message="No portfolio_fund_id provided for transaction import",
                details={'user_message': 'No portfolio_fund_id provided'},
                http_status=400
            )
            return jsonify(response), status
        
        if not file.filename.endswith('.csv'):
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message="Invalid file format for transaction import",
                details={'user_message': 'File must be CSV'},
                http_status=400
            )
            return jsonify(response), status
        
        try:
            # Read and decode the file content
            file_content = file.read()
            decoded_content = file_content.decode('utf-8')
            
            # Check if the file has the correct headers
            first_line = decoded_content.split('\n')[0].strip()
            expected_headers = {'date', 'type', 'shares', 'cost_per_share'}
            found_headers = set(h.strip() for h in first_line.split(','))
            
            if not expected_headers.issubset(found_headers):
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.TRANSACTION,
                    message="Invalid CSV format for transaction import",
                    details={
                        'user_message': 'Invalid CSV format',
                        'expected_headers': list(expected_headers),
                        'found_headers': list(found_headers)
                    },
                    http_status=400
                )
                return jsonify(response), status
            
            portfolio_fund = PortfolioFund.query.get(portfolio_fund_id)
            if not portfolio_fund:
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.TRANSACTION,
                    message=f"Portfolio-fund relationship not found for ID {portfolio_fund_id}",
                    details={'user_message': 'Invalid portfolio-fund relationship'},
                    http_status=400
                )
                return jsonify(response), status
            
            count = DeveloperService.import_transactions_csv(file_content, portfolio_fund_id)
            
            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.TRANSACTION,
                message=f"Successfully imported {count} transactions",
                details={
                    'transaction_count': count,
                    'portfolio_name': portfolio_fund.portfolio.name,
                    'fund_name': portfolio_fund.fund.name
                }
            )

            return jsonify({
                'message': f'Successfully imported {count} transactions',
                'portfolio_name': portfolio_fund.portfolio.name,
                'fund_name': portfolio_fund.fund.name
            })
        except ValueError as e:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=str(e),
                details={'user_message': str(e)},
                http_status=400
            )
            return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Unexpected error during transaction import: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@developer.route('/fund-price', methods=['POST'])
@track_request
def set_fund_price():
    try:
        data = request.json
        result = DeveloperService.set_fund_price(
            data['fund_id'],
            data['price'],
            datetime.strptime(data.get('date', None), '%Y-%m-%d').date() if data.get('date') else None
        )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully set fund price",
            details={
                'fund_id': result['fund_id'],
                'price': result['price'],
                'date': result['date']
            }
        )

        return jsonify(result)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error setting fund price: {str(e)}",
            details={
                'request_data': data,
                'error': str(e)
            },
            http_status=400
        )
        return jsonify(response), status

@developer.route('/csv-template', methods=['GET'])
@track_request
def get_csv_template():
    try:
        template = DeveloperService.get_csv_template()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully retrieved CSV template",
            details={
                'template': template
            }
        )

        return jsonify(template)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving CSV template: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@developer.route('/fund-price-template', methods=['GET'])
@track_request
def get_fund_price_template():
    try:
        template = DeveloperService.get_fund_price_csv_template()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully retrieved fund price CSV template",
            details={
                'template': template
            }
        )

        return jsonify(template)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving fund price CSV template: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@developer.route('/import-fund-prices', methods=['POST'])
@track_request
def import_fund_prices():
    try:
        if 'file' not in request.files:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="No file provided for fund price import",
                details={'user_message': 'No file provided'},
                http_status=400
            )
            return jsonify(response), status
            
        file = request.files['file']
        fund_id = request.form.get('fund_id')
        
        if not fund_id:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="No fund_id provided for fund price import",
                details={'user_message': 'No fund_id provided'},
                http_status=400
            )
            return jsonify(response), status
        
        if not file.filename.endswith('.csv'):
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Invalid file format for fund price import",
                details={'user_message': 'File must be CSV'},
                http_status=400
            )
            return jsonify(response), status
        
        try:
            # Read and decode the file content
            file_content = file.read()
            decoded_content = file_content.decode('utf-8')
            
            # Check if the file has the correct headers
            first_line = decoded_content.split('\n')[0].strip()
            expected_headers = {'date', 'price'}
            found_headers = set(h.strip() for h in first_line.split(','))
            
            if not expected_headers.issubset(found_headers):
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.SYSTEM,
                    message="Invalid CSV format for fund price import",
                    details={
                        'user_message': 'Invalid CSV format',
                        'expected_headers': list(expected_headers),
                        'found_headers': list(found_headers)
                    },
                    http_status=400
                )
                return jsonify(response), status
            
            # Check if this is a transaction file
            if 'type' in found_headers and 'shares' in found_headers:
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.SYSTEM,
                    message="This appears to be a transaction file. Please use the 'Import Transactions' section above to import transactions.",
                    details={'user_message': 'This appears to be a transaction file'},
                    http_status=400
                )
                return jsonify(response), status
            
            count = DeveloperService.import_fund_prices_csv(file_content, fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.SYSTEM,
                message=f"Successfully imported {count} fund prices",
                details={
                    'fund_id': fund_id,
                    'price_count': count
                }
            )

            return jsonify({'message': f'Successfully imported {count} fund prices'})
        except ValueError as e:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=str(e),
                details={'user_message': str(e)},
                http_status=400
            )
            return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Unexpected error during fund price import: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@developer.route('/system-settings/logging', methods=['GET'])
@track_request
def get_logging_settings():
    try:
        settings = {
            'enabled': SystemSetting.get_value(SystemSettingKey.LOGGING_ENABLED, 'true').lower() == 'true',
            'level': SystemSetting.get_value(SystemSettingKey.LOGGING_LEVEL, LogLevel.INFO.value)
        }
        return jsonify(settings)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving logging settings: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@developer.route('/system-settings/logging', methods=['PUT'])
@track_request
def update_logging_settings():
    try:
        data = request.json
        enabled_setting = SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_ENABLED).first()
        if not enabled_setting:
            enabled_setting = SystemSetting(key=SystemSettingKey.LOGGING_ENABLED)
        enabled_setting.value = str(data['enabled']).lower()

        level_setting = SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_LEVEL).first()
        if not level_setting:
            level_setting = SystemSetting(key=SystemSettingKey.LOGGING_LEVEL)
        level_setting.value = data['level']

        db.session.add(enabled_setting)
        db.session.add(level_setting)
        db.session.commit()

        return jsonify({
            'enabled': enabled_setting.value.lower() == 'true',
            'level': level_setting.value
        })
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error updating logging settings: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status

@developer.route('/logs', methods=['GET'])
@track_request
def get_logs():
    try:
        # Get filter parameters
        levels = request.args.get('level')
        categories = request.args.get('category')
        start_date = request.args.get('start_date')
        end_date = request.args.get('end_date')
        source = request.args.get('source')
        
        # Build query
        query = Log.query
        
        if levels:
            levels_list = levels.split(',')
            level_filters = [Log.level == LogLevel(lvl) for lvl in levels_list]
            query = query.filter(db.or_(*level_filters))
        if categories:
            category_list = categories.split(',')
            category_filters = [Log.category == LogCategory(cat) for cat in category_list]
            query = query.filter(db.or_(*category_filters))
        if start_date:
            # Parse ISO timestamp string (already in UTC)
            start_datetime = datetime.fromisoformat(start_date.replace('Z', '+00:00'))
            query = query.filter(Log.timestamp >= start_datetime)
        if end_date:
            # Parse ISO timestamp string (already in UTC)
            end_datetime = datetime.fromisoformat(end_date.replace('Z', '+00:00'))
            query = query.filter(Log.timestamp <= end_datetime)
        if source:
            query = query.filter(Log.source.like(f'%{source}%'))
            
        # Add sorting
        sort_by = request.args.get('sort_by', 'timestamp')
        sort_dir = request.args.get('sort_dir', 'desc')
        
        if sort_dir == 'desc':
            query = query.order_by(getattr(Log, sort_by).desc())
        else:
            query = query.order_by(getattr(Log, sort_by).asc())
            
        # Get paginated results
        page = int(request.args.get('page', 1))
        per_page = int(request.args.get('per_page', 50))
        pagination = query.paginate(page=page, per_page=per_page)
        
        return jsonify({
            'logs': [{
                'id': log.id,
                'timestamp': log.timestamp.strftime('%Y-%m-%d %H:%M:%S.%f'),
                'level': log.level.value,
                'category': log.category.value,
                'message': log.message,
                'details': json.loads(log.details) if log.details else None,
                'source': log.source,
                'request_id': log.request_id,
                'http_status': log.http_status,
                'ip_address': log.ip_address,
                'user_agent': log.user_agent
            } for log in pagination.items],
            'total': pagination.total,
            'pages': pagination.pages,
            'current_page': pagination.page
        })
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving logs: {str(e)}",
            details={'error': str(e)},
            http_status=500
        )
        return jsonify(response), status