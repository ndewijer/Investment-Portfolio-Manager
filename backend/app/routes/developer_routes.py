from flask import Blueprint, jsonify, request
from ..services.developer_service import DeveloperService
from datetime import datetime
from ..models import PortfolioFund

developer = Blueprint('developer', __name__)

@developer.route('/exchange-rate', methods=['GET'])
def get_exchange_rate():
    from_currency = request.args.get('from_currency')
    to_currency = request.args.get('to_currency')
    date = request.args.get('date')
    
    if date:
        date = datetime.strptime(date, '%Y-%m-%d').date()
    else:
        date = datetime.now().date()
        
    exchange_rate = DeveloperService.get_exchange_rate(from_currency, to_currency, date)
    return jsonify(exchange_rate)

@developer.route('/exchange-rate', methods=['POST'])
def set_exchange_rate():
    data = request.json
    result = DeveloperService.set_exchange_rate(
        data['from_currency'],
        data['to_currency'],
        data['rate'],
        datetime.strptime(data.get('date', None), '%Y-%m-%d').date() if data.get('date') else None
    )
    return jsonify(result)

@developer.route('/import-transactions', methods=['POST'])
def import_transactions():
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400
        
    file = request.files['file']
    portfolio_fund_id = request.form.get('fund_id')
    
    if not portfolio_fund_id:
        return jsonify({'error': 'No portfolio_fund_id provided'}), 400
    
    if not file.filename.endswith('.csv'):
        return jsonify({'error': 'File must be CSV'}), 400
    
    try:
        # Read and decode the file content
        file_content = file.read()
        decoded_content = file_content.decode('utf-8')
        print(f"File content preview: {decoded_content[:200]}")
        
        # Check if the file has the correct headers for transactions
        first_line = decoded_content.split('\n')[0].strip()
        expected_headers = {'date', 'type', 'shares', 'cost_per_share'}
        found_headers = set(h.strip() for h in first_line.split(','))
        
        if not expected_headers.issubset(found_headers):
            return jsonify({
                'error': f'Invalid CSV format. Expected headers: {", ".join(expected_headers)}. '
                        f'Found headers: {", ".join(found_headers)}'
            }), 400
        
        portfolio_fund = PortfolioFund.query.get(portfolio_fund_id)
        if not portfolio_fund:
            return jsonify({
                'error': f'Portfolio-fund relationship not found for ID {portfolio_fund_id}. '
                        f'Please ensure you have selected a valid fund for this portfolio.'
            }), 400
        
        count = DeveloperService.import_transactions_csv(file_content, portfolio_fund_id)
        
        return jsonify({
            'message': f'Successfully imported {count} transactions',
            'portfolio_name': portfolio_fund.portfolio.name,
            'fund_name': portfolio_fund.fund.name
        })
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except Exception as e:
        return jsonify({'error': f'Unexpected error: {str(e)}'}), 500

@developer.route('/fund-price', methods=['POST'])
def set_fund_price():
    data = request.json
    result = DeveloperService.set_fund_price(
        data['fund_id'],
        data['price'],
        datetime.strptime(data.get('date', None), '%Y-%m-%d').date() if data.get('date') else None
    )
    return jsonify(result)

@developer.route('/csv-template', methods=['GET'])
def get_csv_template():
    return jsonify(DeveloperService.get_csv_template())

@developer.route('/fund-price-template', methods=['GET'])
def get_fund_price_template():
    return jsonify(DeveloperService.get_fund_price_csv_template())

@developer.route('/import-fund-prices', methods=['POST'])
def import_fund_prices():
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400
        
    file = request.files['file']
    fund_id = request.form.get('fund_id')
    
    if not fund_id:
        return jsonify({'error': 'No fund_id provided'}), 400
    
    if not file.filename.endswith('.csv'):
        return jsonify({'error': 'File must be CSV'}), 400
        
    try:
        # Read and decode the file content
        file_content = file.read()
        decoded_content = file_content.decode('utf-8')
        
        # Check if the file has the correct headers for fund prices
        first_line = decoded_content.split('\n')[0].strip()
        expected_headers = {'date', 'price'}
        found_headers = set(h.strip() for h in first_line.split(','))
        
        if not expected_headers.issubset(found_headers):
            return jsonify({
                'error': f'Invalid CSV format. Expected headers: {", ".join(expected_headers)}. '
                        f'Found headers: {", ".join(found_headers)}'
            }), 400
        
        # Check if this is a transaction file
        if 'type' in found_headers and 'shares' in found_headers:
            return jsonify({
                'error': 'This appears to be a transaction file. Please use the "Import Transactions" section above to import transactions.'
            }), 400
        
        count = DeveloperService.import_fund_prices_csv(file_content, fund_id)
        return jsonify({'message': f'Successfully imported {count} fund prices'})
    except ValueError as e:
        return jsonify({'error': str(e)}), 400
    except Exception as e:
        print(f"Unexpected error during fund price import: {str(e)}")
        return jsonify({'error': f'Unexpected error: {str(e)}'}), 500