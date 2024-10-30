from flask import Blueprint, jsonify, request
from flask.views import MethodView
from datetime import datetime
from ..models import Portfolio, PortfolioFund, db
from ..services.portfolio_service import PortfolioService

portfolios = Blueprint('portfolios', __name__)

class PortfolioAPI(MethodView):
    def get(self, portfolio_id=None):
        if portfolio_id is None:
            portfolios = Portfolio.query.all()
            return jsonify([{
                'id': p.id,
                'name': p.name,
                'description': p.description,
                'is_archived': p.is_archived
            } for p in portfolios])
        else:
            portfolio = Portfolio.query.get_or_404(portfolio_id)
            if portfolio.is_archived:
                return jsonify({'error': 'Portfolio is archived'}), 404
            
            service = PortfolioService()
            value_data = service._calculate_portfolio_value(portfolio, datetime.now().date())
            return jsonify({
                'id': portfolio.id,
                'name': portfolio.name,
                'description': portfolio.description,
                'is_archived': portfolio.is_archived,
                'totalValue': value_data['total_value'],
                'totalCost': value_data['total_cost']
            })

    def post(self):
        data = request.json
        portfolio = Portfolio(name=data['name'], description=data.get('description', ''))
        db.session.add(portfolio)
        db.session.commit()
        return jsonify({
            'id': portfolio.id,
            'name': portfolio.name,
            'description': portfolio.description
        })

    def put(self, portfolio_id):
        portfolio = Portfolio.query.get_or_404(portfolio_id)
        data = request.json
        portfolio.name = data['name']
        portfolio.description = data.get('description', '')
        db.session.commit()
        return jsonify({
            'id': portfolio.id,
            'name': portfolio.name,
            'description': portfolio.description
        })

    def delete(self, portfolio_id):
        portfolio = Portfolio.query.get_or_404(portfolio_id)
        db.session.delete(portfolio)
        db.session.commit()
        return '', 204

@portfolios.route('/portfolios/<int:portfolio_id>/archive', methods=['POST'])
def archive_portfolio(portfolio_id):
    portfolio = Portfolio.query.get_or_404(portfolio_id)
    portfolio.is_archived = True
    db.session.commit()
    return jsonify({
        'id': portfolio.id,
        'name': portfolio.name,
        'description': portfolio.description,
        'is_archived': portfolio.is_archived
    })

@portfolios.route('/portfolios/<int:portfolio_id>/unarchive', methods=['POST'])
def unarchive_portfolio(portfolio_id):
    portfolio = Portfolio.query.get_or_404(portfolio_id)
    portfolio.is_archived = False
    db.session.commit()
    return jsonify({
        'id': portfolio.id,
        'name': portfolio.name,
        'description': portfolio.description,
        'is_archived': portfolio.is_archived
    })

# Register the views
portfolio_view = PortfolioAPI.as_view('portfolio_api')
portfolios.add_url_rule('/portfolios', defaults={'portfolio_id': None},
                     view_func=portfolio_view, methods=['GET'])
portfolios.add_url_rule('/portfolios', view_func=portfolio_view, methods=['POST'])
portfolios.add_url_rule('/portfolios/<int:portfolio_id>',
                     view_func=portfolio_view, methods=['GET', 'PUT', 'DELETE'])

# Summary and History endpoints
@portfolios.route('/portfolio-summary', methods=['GET'])
@portfolios.route('/portfolio-summary/', methods=['GET'])
def get_portfolio_summary():
    service = PortfolioService()
    return jsonify(service.get_portfolio_summary())

@portfolios.route('/portfolio-history', methods=['GET'])
@portfolios.route('/portfolio-history/', methods=['GET'])
def get_portfolio_history():
    service = PortfolioService()
    return jsonify(service.get_portfolio_history())

@portfolios.route('/portfolio-funds', methods=['GET', 'POST'])
@portfolios.route('/portfolio-funds/', methods=['GET', 'POST'])
def handle_portfolio_funds():
    if request.method == 'GET':
        portfolio_id = request.args.get('portfolio_id')
        service = PortfolioService()
        
        if portfolio_id:
            portfolio_funds = service.get_portfolio_funds(portfolio_id)
        else:
            portfolio_funds = service.get_all_portfolio_funds()
            
        return jsonify(portfolio_funds)
    else:
        data = request.json
        portfolio_fund = PortfolioFund(
            portfolio_id=data['portfolio_id'],
            fund_id=data['fund_id']
        )
        db.session.add(portfolio_fund)
        db.session.commit()
        return jsonify({
            'id': portfolio_fund.id,
            'portfolio_id': portfolio_fund.portfolio_id,
            'fund_id': portfolio_fund.fund_id
        })

# Add this new route
@portfolios.route('/portfolios/<int:portfolio_id>/fund-history', methods=['GET'])
def get_portfolio_fund_history(portfolio_id):
    service = PortfolioService()
    history = service.get_portfolio_fund_history(portfolio_id)
    return jsonify(history)