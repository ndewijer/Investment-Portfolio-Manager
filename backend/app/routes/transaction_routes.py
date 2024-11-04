from flask import Blueprint, jsonify, request
from ..models import Transaction, db, LogLevel, LogCategory
from ..services.transaction_service import TransactionService
from ..services.logging_service import logger, track_request

transactions = Blueprint('transactions', __name__)

@transactions.route('/transactions', methods=['GET'])
@track_request
def get_transactions():
    try:
        portfolio_id = request.args.get('portfolio_id')
        service = TransactionService()
        
        if portfolio_id:
            transactions = service.get_portfolio_transactions(portfolio_id)
        else:
            transactions = service.get_all_transactions()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message="Successfully retrieved transactions",
            details={
                'portfolio_id': portfolio_id,
                'transaction_count': len(transactions)
            }
        )
            
        return jsonify(transactions)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error retrieving transactions: {str(e)}",
            details={
                'portfolio_id': portfolio_id,
                'error': str(e)
            },
            http_status=500
        )
        return jsonify(response), status

@transactions.route('/transactions', methods=['POST'])
@track_request
def create_transaction():
    try:
        data = request.json
        service = TransactionService()
        transaction = service.create_transaction(data)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully created transaction",
            details={
                'transaction_id': transaction.id,
                'type': transaction.type,
                'shares': transaction.shares,
                'cost_per_share': transaction.cost_per_share
            }
        )

        return jsonify(service.format_transaction(transaction))
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error creating transaction: {str(e)}",
            details={
                'user_message': 'Error creating transaction',
                'error': str(e),
                'request_data': data
            },
            http_status=400
        )
        return jsonify(response), status

@transactions.route('/transactions/<string:transaction_id>', methods=['GET'])
@track_request
def get_transaction(transaction_id):
    try:
        transaction = Transaction.query.get_or_404(transaction_id)
        service = TransactionService()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully retrieved transaction {transaction_id}",
            details={'transaction_type': transaction.type}
        )

        return jsonify(service.format_transaction(transaction))
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error retrieving transaction: {str(e)}",
            details={
                'transaction_id': transaction_id,
                'error': str(e)
            },
            http_status=404
        )
        return jsonify(response), status

@transactions.route('/transactions/<string:transaction_id>', methods=['PUT'])
@track_request
def update_transaction(transaction_id):
    try:
        data = request.json
        service = TransactionService()
        transaction = service.update_transaction(transaction_id, data)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully updated transaction {transaction_id}",
            details={
                'type': transaction.type,
                'shares': transaction.shares,
                'cost_per_share': transaction.cost_per_share
            }
        )

        return jsonify(service.format_transaction(transaction))
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error updating transaction: {str(e)}",
            details={
                'transaction_id': transaction_id,
                'error': str(e),
                'request_data': data
            },
            http_status=400
        )
        return jsonify(response), status

@transactions.route('/transactions/<string:transaction_id>', methods=['DELETE'])
@track_request
def delete_transaction(transaction_id):
    try:
        transaction = Transaction.query.get_or_404(transaction_id)
        
        # Store transaction details for logging before deletion
        transaction_details = {
            'type': transaction.type,
            'shares': transaction.shares,
            'cost_per_share': transaction.cost_per_share,
            'date': transaction.date.isoformat()
        }
        
        db.session.delete(transaction)
        db.session.commit()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.TRANSACTION,
            message=f"Successfully deleted transaction {transaction_id}",
            details=transaction_details
        )

        return '', 204
    except Exception as e:
        db.session.rollback()
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Error deleting transaction: {str(e)}",
            details={
                'transaction_id': transaction_id,
                'error': str(e)
            },
            http_status=400
        )
        return jsonify(response), status