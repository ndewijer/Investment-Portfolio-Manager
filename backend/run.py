from flask import Flask, request
from flask_cors import CORS
from app.models import db, Portfolio
from app.routes.portfolio_routes import portfolios
from app.routes.fund_routes import funds
from app.routes.transaction_routes import transactions
from app.routes.developer_routes import developer
from app.seed_data import seed_database
import click
from dotenv import load_dotenv
from werkzeug.middleware.proxy_fix import ProxyFix

load_dotenv()

def create_app():
    app = Flask(__name__)
    app.wsgi_app = ProxyFix(app.wsgi_app)
    
    # Configure CORS with all necessary headers
    CORS(app, 
         resources={r"/*": {
             "origins": ["http://localhost:3000", "http://localhost:3001"],
             "methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
             "allow_headers": [
                 "Content-Type", 
                 "Authorization",
                 "Access-Control-Allow-Methods",
                 "Access-Control-Allow-Headers",
                 "Access-Control-Allow-Origin"
             ],
             "expose_headers": ["Content-Type"],
             "supports_credentials": True,
             "send_wildcard": False
         }})
    
    # Configure Flask to handle trailing slashes
    app.url_map.strict_slashes = False
    
    # Database configuration
    app.config['SQLALCHEMY_DATABASE_URI'] = 'sqlite:///portfolio_manager.db'
    app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
    
    # Initialize extensions
    db.init_app(app)
    
    # Register blueprints
    app.register_blueprint(portfolios, url_prefix='/api')
    app.register_blueprint(funds, url_prefix='/api')
    app.register_blueprint(transactions, url_prefix='/api')
    app.register_blueprint(developer, url_prefix='/api')
    
    # CLI commands
    @app.cli.command("seed-db")
    def seed_db_command():
        """Seed the database with sample data."""
        with app.app_context():
            db.create_all()
            seed_database()
            click.echo("Database seeded successfully!")
    
    return app

app = create_app()

if __name__ == '__main__':
    with app.app_context():
        db.create_all()
        if not Portfolio.query.first():
            seed_database()
            print("Database seeded with initial data.")
    
    app.run(debug=True, port=5000)
