from flask import Blueprint, jsonify, request
from ..models import Transaction, db
from ..services.transaction_service import TransactionService

transactions = Blueprint('transactions', __name__)

@transactions.route('/transactions', methods=['GET'])
def get_transactions():
    portfolio_id = request.args.get('portfolio_id')
    service = TransactionService()
    
    if portfolio_id:
        transactions = service.get_portfolio_transactions(portfolio_id)
    else:
        transactions = service.get_all_transactions()
        
    return jsonify(transactions)

@transactions.route('/transactions', methods=['POST'])
def create_transaction():
    data = request.json
    service = TransactionService()
    transaction = service.create_transaction(data)
    return jsonify(service.format_transaction(transaction))

@transactions.route('/transactions/<int:transaction_id>', methods=['GET'])
def get_transaction(transaction_id):
    transaction = Transaction.query.get_or_404(transaction_id)
    service = TransactionService()
    return jsonify(service.format_transaction(transaction))

@transactions.route('/transactions/<int:transaction_id>', methods=['PUT'])
def update_transaction(transaction_id):
    data = request.json
    service = TransactionService()
    transaction = service.update_transaction(transaction_id, data)
    return jsonify(service.format_transaction(transaction))

@transactions.route('/transactions/<int:transaction_id>', methods=['DELETE'])
def delete_transaction(transaction_id):
    service = TransactionService()
    service.delete_transaction(transaction_id)
    return '', 204