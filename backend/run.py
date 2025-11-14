"""Main application entry point."""

import os

import click
from app.models import LogLevel, Portfolio, SystemSetting, SystemSettingKey, db
from app.routes.developer_routes import developer
from app.routes.dividend_routes import dividends
from app.routes.fund_routes import funds
from app.routes.ibkr_routes import ibkr
from app.routes.portfolio_routes import portfolios
from app.routes.system_routes import system
from app.routes.transaction_routes import transactions
from app.seed_data import seed_database
from app.tasks.ibkr_import import run_automated_ibkr_import
from app.tasks.price_updates import update_all_fund_prices
from apscheduler.schedulers.background import BackgroundScheduler
from apscheduler.triggers.cron import CronTrigger
from dotenv import load_dotenv
from flask import Flask
from flask_cors import CORS
from flask_migrate import Migrate
from werkzeug.middleware.proxy_fix import ProxyFix

load_dotenv()

# Buffer for startup logs (before app context is available)
_startup_logs = []


def log_startup_message(level, category, message, details=None):
    """
    Log a message during startup (before app context is available).

    Prints immediately for console visibility and buffers for database logging.

    Args:
        level: LogLevel enum value
        category: LogCategory enum value
        message: Log message string
        details: Optional dict of additional details
    """
    # Print immediately for console visibility
    print(f"[{level.value.upper()}] [{category.value.upper()}] {message}")
    if details:
        for key, value in details.items():
            print(f"  {key}: {value}")

    # Store for later database logging
    _startup_logs.append(
        {"level": level, "category": category, "message": message, "details": details or {}}
    )


def flush_startup_logs():
    """
    Flush buffered startup logs to database once app context is available.

    Must be called within Flask application context.
    """
    if not _startup_logs:
        return

    from app.services.logging_service import logger

    count = len(_startup_logs)
    for log_entry in _startup_logs:
        try:
            logger.log(
                level=log_entry["level"],
                category=log_entry["category"],
                message=log_entry["message"],
                details=log_entry["details"],
            )
        except Exception as e:
            print(f"[ERROR] Failed to flush startup log to database: {e}")

    _startup_logs.clear()
    print(f"[INFO] Flushed {count} startup log(s) to database")


def get_version():
    """Get application version from VERSION file."""
    version_file = os.path.join(os.path.dirname(__file__), "VERSION")
    with open(version_file) as f:
        return f.read().strip()


def get_or_create_ibkr_encryption_key():
    """
    Get or create IBKR encryption key.

    This has the following priority:
    1. IBKR_ENCRYPTION_KEY environment variable (best practice for migrations)
    2. /data/.ibkr_encryption_key file (auto-generated, persisted)
    3. Generate new key and save to file.

    Returns:
        str: Base64-encoded Fernet encryption key

    Note: This function uses log_startup_message() which prints immediately
    and buffers logs for database insertion once app context is available.
    """
    from app.models import LogCategory
    from cryptography.fernet import Fernet

    # Priority 1: Environment variable (best practice for migrations)
    env_key = os.environ.get("IBKR_ENCRYPTION_KEY")
    if env_key:
        log_startup_message(
            level=LogLevel.INFO,
            category=LogCategory.SECURITY,
            message="Using IBKR encryption key from environment variable",
            details={"source": "IBKR_ENCRYPTION_KEY", "method": "environment_variable"},
        )
        return env_key

    # Priority 2: Persistent key file
    data_dir = os.environ.get("DB_DIR", os.path.join(os.getcwd(), "data/db"))
    key_file = os.path.join(os.path.dirname(data_dir), ".ibkr_encryption_key")

    if os.path.exists(key_file):
        try:
            with open(key_file) as f:
                key = f.read().strip()
            log_startup_message(
                level=LogLevel.INFO,
                category=LogCategory.SECURITY,
                message="Using IBKR encryption key from persistent storage",
                details={"source": key_file, "method": "persistent_file"},
            )
            return key
        except Exception as e:
            log_startup_message(
                level=LogLevel.ERROR,
                category=LogCategory.SECURITY,
                message="Failed to read encryption key file",
                details={"error": str(e), "file": key_file},
            )

    # Priority 3: Generate new key
    new_key = Fernet.generate_key().decode()

    log_startup_message(
        level=LogLevel.WARNING,
        category=LogCategory.SECURITY,
        message="IBKR encryption key auto-generated",
        details={
            "key": new_key,
            "saved_to": key_file,
            "action_required": "Save this key for database migrations",
            "best_practice": "Set IBKR_ENCRYPTION_KEY in .env file",
        },
    )

    # Save to persistent file
    try:
        os.makedirs(os.path.dirname(key_file), exist_ok=True)
        with open(key_file, "w") as f:
            f.write(new_key)
        os.chmod(key_file, 0o600)  # Owner read/write only

        log_startup_message(
            level=LogLevel.INFO,
            category=LogCategory.SECURITY,
            message="Encryption key saved to persistent storage",
            details={"file": key_file, "permissions": "0600"},
        )
    except Exception as e:
        log_startup_message(
            level=LogLevel.ERROR,
            category=LogCategory.SECURITY,
            message="Failed to save encryption key to file",
            details={
                "error": str(e),
                "file": key_file,
                "consequence": "Key will NOT persist across container restarts",
                "solution": "Set IBKR_ENCRYPTION_KEY environment variable",
            },
        )

    return new_key


def create_app(config=None):
    """Create and configure the Flask application.

    Args:
        config (dict, optional): Configuration dictionary to override default config.
            Useful for testing with a separate test database.
    """
    app = Flask(__name__)
    app.config["VERSION"] = get_version()  # Add version to app config
    app.wsgi_app = ProxyFix(app.wsgi_app)

    # Apply custom config first if provided (e.g., for testing)
    if config:
        app.config.update(config)

    # Get database directory from environment variable or use default
    # Skip if already configured (e.g., in test config)
    if "SQLALCHEMY_DATABASE_URI" not in app.config:
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

    # Initialize IBKR encryption key
    app.config["IBKR_ENCRYPTION_KEY"] = get_or_create_ibkr_encryption_key()

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
    migrate = Migrate(app, db)  # noqa: F841

    # Enable foreign key constraints for SQLite
    from sqlalchemy import event
    from sqlalchemy.engine import Engine

    @event.listens_for(Engine, "connect")
    def set_sqlite_pragma(dbapi_conn, connection_record):
        """Enable foreign key constraints for SQLite connections."""
        cursor = dbapi_conn.cursor()
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.close()

    # Register blueprints
    app.register_blueprint(portfolios, url_prefix="/api")
    app.register_blueprint(funds, url_prefix="/api")
    app.register_blueprint(transactions, url_prefix="/api")
    app.register_blueprint(developer, url_prefix="/api")
    app.register_blueprint(dividends, url_prefix="/api")
    app.register_blueprint(ibkr, url_prefix="/api")
    app.register_blueprint(system, url_prefix="/api")

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
            if not SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_ENABLED).first():
                logging_enabled = SystemSetting(key=SystemSettingKey.LOGGING_ENABLED, value="true")
                db.session.add(logging_enabled)

            if not SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_LEVEL).first():
                logging_level = SystemSetting(
                    key=SystemSettingKey.LOGGING_LEVEL, value=LogLevel.ERROR.value
                )
                db.session.add(logging_level)

            db.session.commit()

        # Flush buffered startup logs to database (skip in test mode)
        if not app.config.get("TESTING", False):
            flush_startup_logs()

    # Set up scheduler
    scheduler = BackgroundScheduler()

    # Create wrapper functions that provide app context
    def run_price_updates():
        with app.app_context():
            update_all_fund_prices()

    def run_ibkr_import():
        with app.app_context():
            run_automated_ibkr_import()

    # Schedule the price update task to run at 23:55 local time every weekday
    scheduler.add_job(
        func=run_price_updates,
        trigger=CronTrigger(hour=23, minute=55, day_of_week="mon-fri"),
        id="daily_price_update",
        name="Update all fund prices daily",
        replace_existing=True,
    )

    # Schedule the IBRK Import task to run at 05:05 local time every weekday
    scheduler.add_job(
        func=run_ibkr_import,
        trigger=CronTrigger(hour=5, minute=5, day_of_week="thu-sat"),
        id="weekly_ibkr_import",
        name="Import IBKR transactions weekly",
        replace_existing=True,
    )

    scheduler.start()

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

if __name__ == "__main__":
    """Run the Flask application."""
    with app.app_context():
        db.create_all()
        if not Portfolio.query.first():
            seed_database()
            print("Database seeded with initial data.")

    app.run(debug=True, port=5000)
