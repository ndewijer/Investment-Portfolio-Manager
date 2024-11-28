"""Main application entry point."""

import os

import click
from dotenv import load_dotenv
from flask import Flask
from flask_cors import CORS
from werkzeug.middleware.proxy_fix import ProxyFix
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from flask_migrate import Migrate

from app.models import LogLevel, Portfolio, SystemSetting, SystemSettingKey, db
from app.routes.developer_routes import developer
from app.routes.dividend_routes import dividends
from app.routes.fund_routes import funds
from app.routes.portfolio_routes import portfolios
from app.routes.transaction_routes import transactions
from app.seed_data import seed_database

from app.tasks.price_updates import update_all_fund_prices

load_dotenv()


def create_app():
    """Create and configure the Flask application."""
    app = Flask(__name__)
    app.wsgi_app = ProxyFix(app.wsgi_app)

    # Get database directory from environment variable or use default
    db_dir = os.environ.get("DB_DIR", os.path.join(os.getcwd(), "data/db"))
    print(f"DB_DIR: {db_dir}")
    # Ensure the directory exists
    os.makedirs(db_dir, exist_ok=True)

    # Configure database URI using the DB_DIR
    db_path = os.path.join(db_dir, "portfolio_manager.db")
    app.config["SQLALCHEMY_DATABASE_URI"] = f"sqlite:///{db_path}"
    app.config["SQLALCHEMY_TRACK_MODIFICATIONS"] = False

    # Configure logging directory
    log_dir = os.environ.get("LOG_DIR", os.path.join(os.getcwd(), "data/logs"))
    os.makedirs(log_dir, exist_ok=True)
    app.config["LOG_DIR"] = log_dir

    # Get hostname from environment variable
    frontend_host = os.environ.get("DOMAIN", "*")

    # Configure CORS with all necessary headers
    CORS(
        app,
        resources={
            r"/*": {
                "origins": [
                    f"http://{frontend_host}",  # Production HTTP
                    f"https://{frontend_host}",  # Production HTTPS
                    "http://localhost:3000",  # Development scripts
                    "http://localhost",  # Development docker
                ],
                "methods": ["GET", "POST", "PUT", "DELETE", "OPTIONS"],
                "allow_headers": [
                    "Content-Type",
                    "Authorization",
                    "Access-Control-Allow-Methods",
                    "Access-Control-Allow-Headers",
                    "Access-Control-Allow-Origin",
                    "Access-Control-Allow-Credentials",
                ],
                "expose_headers": ["Content-Type"],
                "supports_credentials": True,
                "send_wildcard": False,
                "max_age": 86400,  # Cache preflight requests for 24 hours
            }
        },
    )

    # Configure Flask to handle trailing slashes
    app.url_map.strict_slashes = False

    # Initialize extensions
    db.init_app(app)
    migrate = Migrate(app, db)

    # Register blueprints
    app.register_blueprint(portfolios, url_prefix="/api")
    app.register_blueprint(funds, url_prefix="/api")
    app.register_blueprint(transactions, url_prefix="/api")
    app.register_blueprint(developer, url_prefix="/api")
    app.register_blueprint(dividends, url_prefix="/api")

    # Create database tables and set default settings
    with app.app_context():
        # Check if database exists by trying to query any table
        try:
            SystemSetting.query.first()
        except Exception:
            # If query fails, database doesn't exist yet
            db.create_all()
        else:
            # Database exists, set default settings if needed
            if not SystemSetting.query.filter_by(
                key=SystemSettingKey.LOGGING_ENABLED
            ).first():
                logging_enabled = SystemSetting(
                    key=SystemSettingKey.LOGGING_ENABLED, value="true"
                )
                db.session.add(logging_enabled)

            if not SystemSetting.query.filter_by(
                key=SystemSettingKey.LOGGING_LEVEL
            ).first():
                logging_level = SystemSetting(
                    key=SystemSettingKey.LOGGING_LEVEL, value=LogLevel.ERROR.value
                )
                db.session.add(logging_level)

            db.session.commit()

    # Set up scheduler
    scheduler = BackgroundScheduler()

    # Create a wrapper function that provides app context
    def run_price_updates():
        with app.app_context():
            update_all_fund_prices()

    # Schedule the price update task to run at 23:55 local time every weekday
    scheduler.add_job(
        func=run_price_updates,  # Use the wrapper function instead
        trigger=CronTrigger(hour=23, minute=55, day_of_week="mon-fri"),
        id="daily_price_update",
        name="Update all fund prices daily",
        replace_existing=True,
    )

    scheduler.start()

    # CLI commands
    @app.cli.command("seed-db")
    def seed_db_command():
        """Seed the database with sample data."""
        with app.app_context():
            #db.create_all()
            seed_database()
            click.echo("Database seeded successfully!")

    return app


app = create_app()

if __name__ == "__main__":
    """Run the Flask application."""
    with app.app_context():
        db.create_all()
        if not Portfolio.query.first():
            seed_database()
            print("Database seeded with initial data.")

    app.run(debug=True, port=5000)
