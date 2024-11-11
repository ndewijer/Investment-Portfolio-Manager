from flask import Flask, request
from flask_cors import CORS
from app.models import db, Portfolio, SystemSetting, SystemSettingKey, LogLevel, Fund
from app.routes.portfolio_routes import portfolios
from app.routes.fund_routes import funds
from app.routes.transaction_routes import transactions
from app.routes.developer_routes import developer
from app.routes.dividend_routes import dividends
from app.seed_data import seed_database
import click
from dotenv import load_dotenv
from werkzeug.middleware.proxy_fix import ProxyFix
from app.services.price_update_service import TodayPriceService, HistoricalPriceService
import os

load_dotenv()

def create_app():
    app = Flask(__name__)
    app.wsgi_app = ProxyFix(app.wsgi_app)
    
    # Get database directory from environment variable or use default
    db_dir = os.environ.get('DB_DIR', os.path.join(os.getcwd(), 'data/db'))
    print(f"DB_DIR: {db_dir}")
    # Ensure the directory exists
    os.makedirs(db_dir, exist_ok=True)
    
    # Configure database URI using the DB_DIR
    db_path = os.path.join(db_dir, 'portfolio_manager.db')
    app.config['SQLALCHEMY_DATABASE_URI'] = f'sqlite:///{db_path}'
    app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
    
    # Configure logging directory
    log_dir = os.environ.get('LOG_DIR', os.path.join(os.getcwd(), 'data/logs'))
    os.makedirs(log_dir, exist_ok=True)
    app.config['LOG_DIR'] = log_dir
    
    # Get hostname from environment variable
    frontend_host = os.environ.get('FRONTEND_HOST', '*')
    
    # Configure CORS with all necessary headers
    CORS(app, 
         resources={r"/*": {
             "origins": [
                 f"http://{frontend_host}",  # Production HTTP
                 f"https://{frontend_host}",  # Production HTTPS
                 "http://localhost:3000",     # Development
             ],
             "methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
             "allow_headers": [
                 "Content-Type", 
                 "Authorization",
                 "Access-Control-Allow-Methods",
                 "Access-Control-Allow-Headers",
                 "Access-Control-Allow-Origin",
                 "Access-Control-Allow-Credentials"
             ],
             "expose_headers": ["Content-Type"],
             "supports_credentials": True,
             "send_wildcard": False,
             "max_age": 86400  # Cache preflight requests for 24 hours
         }})
    
    # Configure Flask to handle trailing slashes
    app.url_map.strict_slashes = False
    
    # Initialize extensions
    db.init_app(app)
    
    # Register blueprints
    app.register_blueprint(portfolios, url_prefix='/api')
    app.register_blueprint(funds, url_prefix='/api')
    app.register_blueprint(transactions, url_prefix='/api')
    app.register_blueprint(developer, url_prefix='/api')
    app.register_blueprint(dividends, url_prefix='/api')
    
    # Create database tables and set default settings
    with app.app_context():
        db.create_all()
        
        # Set default system settings if they don't exist
        if not SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_ENABLED).first():
            logging_enabled = SystemSetting(
                key=SystemSettingKey.LOGGING_ENABLED,
                value='true'
            )
            db.session.add(logging_enabled)

        if not SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_LEVEL).first():
            logging_level = SystemSetting(
                key=SystemSettingKey.LOGGING_LEVEL,
                value=LogLevel.ERROR.value
            )
            db.session.add(logging_level)

        db.session.commit()

    # CLI commands
    @app.cli.command("seed-db")
    def seed_db_command():
        """Seed the database with sample data."""
        with app.app_context():
            db.create_all()
            seed_database()
            click.echo("Database seeded successfully!")
    
    @app.cli.command("update-prices")
    def update_prices_command():
        """Update all fund prices."""
        with app.app_context():
            funds = Fund.query.filter(Fund.symbol.isnot(None)).all()
            results = []
            
            for fund in funds:
                # Update today's price for funds
                if fund.investment_type == 'fund':
                    today_response, _ = TodayPriceService.update_todays_price(fund.id)
                
                # Update historical prices for all
                hist_response, _ = HistoricalPriceService.update_historical_prices(fund.id)
                
                results.append({
                    'fund_id': fund.id,
                    'name': fund.name,
                    'success': hist_response.get('status') == 'success'
                })
            
            click.echo(f"Updated prices for {len(results)} funds")
            for result in results:
                status = "✓" if result['success'] else "✗"
                click.echo(f"{status} {result['name']}")
    
    return app

app = create_app()

if __name__ == '__main__':
    with app.app_context():
        db.create_all()
        if not Portfolio.query.first():
            seed_database()
            print("Database seeded with initial data.")
    
    app.run(debug=True, port=5000)
