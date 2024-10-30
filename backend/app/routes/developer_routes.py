from flask import Blueprint, jsonify, request
from ..services.developer_service import DeveloperService
from datetime import datetime

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
    portfolio_id = request.form.get('portfolio_id')
    fund_id = request.form.get('fund_id')
    
    if not file.filename.endswith('.csv'):
        return jsonify({'error': 'File must be CSV'}), 400
        
    try:
        count = DeveloperService.import_transactions_csv(file.read(), portfolio_id, fund_id)
        return jsonify({'message': f'Successfully imported {count} transactions'})
    except ValueError as e:
        return jsonify({'error': str(e)}), 400

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
    
    if not file.filename.endswith('.csv'):
        return jsonify({'error': 'File must be CSV'}), 400
        
    try:
        count = DeveloperService.import_fund_prices_csv(file.read(), fund_id)
        return jsonify({'message': f'Successfully imported {count} fund prices'})
    except ValueError as e:
        return jsonify({'error': str(e)}), 400