"""
Developer-focused API routes for system maintenance and monitoring.

This module provides routes for:
- Exchange rate management
- Fund price management
- System logging configuration
- CSV template generation
- Transaction imports
"""

import json
from datetime import datetime

from flask import Blueprint, jsonify, request

from ..models import (
    Fund,
    Log,
    LogCategory,
    LogLevel,
    PortfolioFund,
    SystemSetting,
    SystemSettingKey,
    db,
)
from ..services.developer_service import DeveloperService
from ..services.logging_service import logger, track_request

developer = Blueprint("developer", __name__)


@developer.route("/exchange-rate", methods=["GET"])
@track_request
def get_exchange_rate():
    """
    Get exchange rate for a currency pair on a specific date.

    Query Parameters:
        from_currency (str): Source currency code
        to_currency (str): Target currency code
        date (str): Optional date in YYYY-MM-DD format

    Returns:
        JSON response containing exchange rate details or error message
    """
    try:
        from_currency = request.args.get("from_currency")
        to_currency = request.args.get("to_currency")
        date = request.args.get("date")

        if date:
            date = datetime.strptime(date, "%Y-%m-%d").date()
        else:
            date = datetime.now().date()

        exchange_rate = DeveloperService.get_exchange_rate(from_currency, to_currency, date)

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully retrieved exchange rate",
            details={
                "from_currency": from_currency,
                "to_currency": to_currency,
                "date": date.isoformat(),
                "rate": exchange_rate["rate"] if exchange_rate else None,
            },
        )

        return jsonify(exchange_rate)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving exchange rate: {e!s}",
            details={
                "from_currency": from_currency,
                "to_currency": to_currency,
                "date": date.isoformat() if date else None,
                "error": str(e),
            },
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/exchange-rate", methods=["POST"])
@track_request
def set_exchange_rate():
    """
    Set exchange rate for a currency pair.

    Request Body:
        from_currency (str): Source currency code
        to_currency (str): Target currency code
        rate (float): Exchange rate value
        date (str, optional): Date in YYYY-MM-DD format

    Returns:
        JSON response containing exchange rate details or error message
    """
    try:
        data = request.json

        # Validate required fields
        required_fields = ["from_currency", "to_currency", "rate"]
        for field in required_fields:
            if field not in data:
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.SYSTEM,
                    message=f"Missing required field: {field}",
                    details={
                        "user_message": f"Missing required field: {field}",
                        "request_data": data,
                    },
                    http_status=400,
                )
                return jsonify(response), status

        # Validate currency codes
        valid_currencies = {"USD", "EUR", "GBP", "JPY"}  # Add more as needed
        if data["from_currency"] not in valid_currencies:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=f"Invalid from_currency: {data['from_currency']}",
                details={
                    "user_message": "Invalid currency code",
                    "valid_currencies": list(valid_currencies),
                },
                http_status=400,
            )
            return jsonify(response), status

        result = DeveloperService.set_exchange_rate(
            data["from_currency"],
            data["to_currency"],
            data["rate"],
            (
                datetime.strptime(data.get("date", None), "%Y-%m-%d").date()
                if data.get("date")
                else None
            ),
        )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully set exchange rate",
            details=result,
        )

        return jsonify(result)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error setting exchange rate: {e!s}",
            details={"request_data": data, "error": str(e)},
            http_status=400,
        )
        return jsonify(response), status


@developer.route("/import-transactions", methods=["POST"])
@track_request
def import_transactions():
    """
    Import transactions from a CSV file.

    Form Data:
        file: CSV file containing transaction data
        fund_id: Portfolio-Fund relationship ID

    CSV Format:
        date,type,shares,cost_per_share

    Returns:
        JSON response with import results or error message
    """
    try:
        if "file" not in request.files:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message="No file provided for transaction import",
                details={"user_message": "No file provided"},
                http_status=400,
            )
            return jsonify(response), status

        file = request.files["file"]
        portfolio_fund_id = request.form.get("fund_id")

        if not portfolio_fund_id:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message="No portfolio_fund_id provided for transaction import",
                details={"user_message": "No portfolio_fund_id provided"},
                http_status=400,
            )
            return jsonify(response), status

        if not file.filename.endswith(".csv"):
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message="Invalid file format for transaction import",
                details={"user_message": "File must be CSV"},
                http_status=400,
            )
            return jsonify(response), status

        try:
            # Read and decode the file content
            file_content = file.read()
            decoded_content = file_content.decode("utf-8-sig")

            # Check if the file has the correct headers
            first_line = decoded_content.split("\n")[0].strip()
            expected_headers = {"date", "type", "shares", "cost_per_share"}
            found_headers = {h.strip() for h in first_line.split(",")}

            if not expected_headers.issubset(found_headers):
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.TRANSACTION,
                    message="Invalid CSV format for transaction import",
                    details={
                        "user_message": "Invalid CSV format",
                        "expected_headers": list(expected_headers),
                        "found_headers": list(found_headers),
                    },
                    http_status=400,
                )
                return jsonify(response), status

            portfolio_fund = db.session.get(PortfolioFund, portfolio_fund_id)
            if not portfolio_fund:
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.TRANSACTION,
                    message=f"Portfolio-fund relationship not found for ID {portfolio_fund_id}",
                    details={"user_message": "Invalid portfolio-fund relationship"},
                    http_status=400,
                )
                return jsonify(response), status

            # Re-encode the content without BOM for the service
            file_content = decoded_content.encode("utf-8")
            count = DeveloperService.import_transactions_csv(file_content, portfolio_fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.TRANSACTION,
                message=f"Successfully imported {count} transactions",
                details={
                    "transaction_count": count,
                    "portfolio_name": portfolio_fund.portfolio.name,
                    "fund_name": portfolio_fund.fund.name,
                },
            )

            return jsonify(
                {
                    "message": f"Successfully imported {count} transactions",
                    "portfolio_name": portfolio_fund.portfolio.name,
                    "fund_name": portfolio_fund.fund.name,
                }
            )
        except ValueError as e:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.TRANSACTION,
                message=str(e),
                details={"user_message": str(e)},
                http_status=400,
            )
            return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.TRANSACTION,
            message=f"Unexpected error during transaction import: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/fund-price", methods=["POST"])
@track_request
def set_fund_price():
    """
    Set price for a specific fund.

    Request Body:
        fund_id (str): Fund identifier
        price (float): Fund price value
        date (str, optional): Date in YYYY-MM-DD format

    Returns:
        JSON response containing fund price details or error message
    """
    try:
        data = request.json

        # Validate required fields
        required_fields = ["fund_id", "price"]
        for field in required_fields:
            if field not in data:
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.SYSTEM,
                    message=f"Missing required field: {field}",
                    details={
                        "user_message": f"Missing required field: {field}",
                        "request_data": data,
                    },
                    http_status=400,
                )
                return jsonify(response), status

        # Validate fund_id exists
        fund = db.session.get(Fund, data["fund_id"])
        if not fund:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=f"Invalid fund_id: {data['fund_id']}",
                details={"user_message": "Fund not found", "fund_id": data["fund_id"]},
                http_status=400,
            )
            return jsonify(response), status

        result = DeveloperService.set_fund_price(
            data["fund_id"],
            data["price"],
            (
                datetime.strptime(data.get("date", None), "%Y-%m-%d").date()
                if data.get("date")
                else None
            ),
        )

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully set fund price",
            details=result,
        )

        return jsonify(result)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error setting fund price: {e!s}",
            details={"request_data": data, "error": str(e)},
            http_status=400,
        )
        return jsonify(response), status


@developer.route("/csv-template", methods=["GET"])
@track_request
def get_csv_template():
    """
    Get CSV template for transaction imports.

    Returns:
        JSON response containing CSV template structure
    """
    try:
        template = DeveloperService.get_csv_template()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully retrieved CSV template",
            details={"template": template},
        )

        return jsonify(template)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving CSV template: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/fund-price-template", methods=["GET"])
@track_request
def get_fund_price_template():
    """
    Get CSV template for fund price imports.

    Returns:
        JSON response containing fund price CSV template structure
    """
    try:
        template = DeveloperService.get_fund_price_csv_template()

        logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="Successfully retrieved fund price CSV template",
            details={"template": template},
        )

        return jsonify(template)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving fund price CSV template: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/import-fund-prices", methods=["POST"])
@track_request
def import_fund_prices():
    """
    Import fund prices from a CSV file.

    Request Body:
        file: CSV file containing fund price data
        fund_id: Fund identifier
    """
    try:
        if "file" not in request.files:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="No file provided for fund price import",
                details={"user_message": "No file provided"},
                http_status=400,
            )
            return jsonify(response), status

        file = request.files["file"]
        fund_id = request.form.get("fund_id")

        if not fund_id:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="No fund_id provided for fund price import",
                details={"user_message": "No fund_id provided"},
                http_status=400,
            )
            return jsonify(response), status

        if not file.filename.endswith(".csv"):
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message="Invalid file format for fund price import",
                details={"user_message": "File must be CSV"},
                http_status=400,
            )
            return jsonify(response), status

        try:
            # Read and decode the file content
            file_content = file.read()
            decoded_content = file_content.decode("utf-8-sig")

            # Check if the file has the correct headers
            first_line = decoded_content.split("\n")[0].strip()
            expected_headers = {"date", "price"}
            found_headers = {h.strip() for h in first_line.split(",")}

            if not expected_headers.issubset(found_headers):
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.SYSTEM,
                    message="Invalid CSV format for fund price import",
                    details={
                        "user_message": "Invalid CSV format",
                        "expected_headers": list(expected_headers),
                        "found_headers": list(found_headers),
                    },
                    http_status=400,
                )
                return jsonify(response), status

            # Check if this is a transaction file
            if "type" in found_headers and "shares" in found_headers:
                response, status = logger.log(
                    level=LogLevel.ERROR,
                    category=LogCategory.SYSTEM,
                    message="This appears to be a transaction file. Please use the 'Import Transactions' section above to import transactions.",  # noqa: E501
                    details={"user_message": "This appears to be a transaction file"},
                    http_status=400,
                )
                return jsonify(response), status

            count = DeveloperService.import_fund_prices_csv(file_content, fund_id)

            logger.log(
                level=LogLevel.INFO,
                category=LogCategory.SYSTEM,
                message=f"Successfully imported {count} fund prices",
                details={"fund_id": fund_id, "price_count": count},
            )

            return jsonify({"message": f"Successfully imported {count} fund prices"})
        except ValueError as e:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=str(e),
                details={"user_message": str(e)},
                http_status=400,
            )
            return jsonify(response), status
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Unexpected error during fund price import: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/system-settings/logging", methods=["GET"])
@track_request
def get_logging_settings():
    """
    Get current logging configuration settings.

    Returns:
        JSON response containing logging enabled state and level
    """
    try:
        settings = {
            "enabled": SystemSetting.get_value(SystemSettingKey.LOGGING_ENABLED, "true").lower()
            == "true",
            "level": SystemSetting.get_value(SystemSettingKey.LOGGING_LEVEL, LogLevel.INFO.value),
        }
        return jsonify(settings)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving logging settings: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/system-settings/logging", methods=["PUT"])
@track_request
def update_logging_settings():
    """
    Update logging configuration settings.

    Request Body:
        enabled (bool): Enable/disable logging
        level (str): Logging level

    Returns:
        JSON response containing updated logging settings
    """
    try:
        data = request.json
        enabled_setting = SystemSetting.query.filter_by(
            key=SystemSettingKey.LOGGING_ENABLED
        ).first()
        if not enabled_setting:
            enabled_setting = SystemSetting(key=SystemSettingKey.LOGGING_ENABLED)
        enabled_setting.value = str(data["enabled"]).lower()

        level_setting = SystemSetting.query.filter_by(key=SystemSettingKey.LOGGING_LEVEL).first()
        if not level_setting:
            level_setting = SystemSetting(key=SystemSettingKey.LOGGING_LEVEL)
        level_setting.value = data["level"]

        db.session.add(enabled_setting)
        db.session.add(level_setting)
        db.session.commit()

        return jsonify(
            {
                "enabled": enabled_setting.value.lower() == "true",
                "level": level_setting.value,
            }
        )
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error updating logging settings: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/logs", methods=["GET"])
@track_request
def get_logs():
    """
    Retrieve filtered and paginated system logs.

    Query Parameters:
        level (str, optional): Comma-separated list of log levels
        category (str, optional): Comma-separated list of log categories
        start_date (str, optional): Start date in ISO format
        end_date (str, optional): End date in ISO format
        source (str, optional): Source filter
        sort_by (str, optional): Field to sort by (default: timestamp)
        sort_dir (str, optional): Sort direction (asc/desc, default: desc)
        page (int, optional): Page number (default: 1)
        per_page (int, optional): Items per page (default: 50)

    Returns:
        JSON response containing paginated logs and metadata
    """
    try:
        # Get filter parameters
        levels = request.args.get("level")
        categories = request.args.get("category")
        start_date = request.args.get("start_date")
        end_date = request.args.get("end_date")
        source = request.args.get("source")

        # Build query
        query = Log.query

        if levels:
            levels_list = levels.split(",")
            level_filters = [Log.level == LogLevel(lvl) for lvl in levels_list]
            query = query.filter(db.or_(*level_filters))
        if categories:
            category_list = categories.split(",")
            category_filters = [Log.category == LogCategory(cat) for cat in category_list]
            query = query.filter(db.or_(*category_filters))
        if start_date:
            # Parse ISO timestamp string (already in UTC)
            start_datetime = datetime.fromisoformat(start_date.replace("Z", "+00:00"))
            query = query.filter(Log.timestamp >= start_datetime)
        if end_date:
            # Parse ISO timestamp string (already in UTC)
            end_datetime = datetime.fromisoformat(end_date.replace("Z", "+00:00"))
            query = query.filter(Log.timestamp <= end_datetime)
        if source:
            query = query.filter(Log.source.like(f"%{source}%"))

        # Add sorting
        sort_by = request.args.get("sort_by", "timestamp")
        sort_dir = request.args.get("sort_dir", "desc")

        if sort_dir == "desc":
            query = query.order_by(getattr(Log, sort_by).desc())
        else:
            query = query.order_by(getattr(Log, sort_by).asc())

        # Get paginated results
        page = int(request.args.get("page", 1))
        per_page = int(request.args.get("per_page", 50))
        pagination = query.paginate(page=page, per_page=per_page)

        return jsonify(
            {
                "logs": [
                    {
                        "id": log.id,
                        "timestamp": log.timestamp.strftime("%Y-%m-%d %H:%M:%S.%f"),
                        "level": log.level.value,
                        "category": log.category.value,
                        "message": log.message,
                        "details": json.loads(log.details) if log.details else None,
                        "source": log.source,
                        "request_id": log.request_id,
                        "http_status": log.http_status,
                        "ip_address": log.ip_address,
                        "user_agent": log.user_agent,
                    }
                    for log in pagination.items
                ],
                "total": pagination.total,
                "pages": pagination.pages,
                "current_page": pagination.page,
            }
        )
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving logs: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/fund-price/<string:fund_id>", methods=["GET"])
@track_request
def get_fund_price(fund_id):
    """
    Get fund price for a specific fund and date.

    Path Parameters:
        fund_id (str): Fund identifier

    Query Parameters:
        date (str, optional): Date in YYYY-MM-DD format

    Returns:
        JSON response containing fund price details or error message
    """
    try:
        date = request.args.get("date")
        if date:
            date = datetime.strptime(date, "%Y-%m-%d").date()
        else:
            date = datetime.now().date()

        result = DeveloperService.get_fund_price(fund_id, date)
        if not result:
            response, status = logger.log(
                level=LogLevel.ERROR,
                category=LogCategory.SYSTEM,
                message=f"Fund price not found",
                details={"fund_id": fund_id, "date": date.isoformat()},
                http_status=404,
            )
            return jsonify(response), status

        return jsonify(result)
    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error retrieving fund price: {e!s}",
            details={
                "fund_id": fund_id,
                "date": date.isoformat() if date else None,
                "error": str(e),
            },
            http_status=500,
        )
        return jsonify(response), status


@developer.route("/logs/clear", methods=["POST"])
@track_request
def clear_logs():
    """
    Clear all system logs from the database.

    Returns:
        JSON response confirming logs were cleared or error message
    """
    try:
        # Delete all logs from the database
        Log.query.delete()
        db.session.commit()

        response, status = logger.log(
            level=LogLevel.INFO,
            category=LogCategory.SYSTEM,
            message="All logs cleared by user",
            source="clear_logs",
            http_status=200,
        )

        return jsonify(response), status

    except Exception as e:
        response, status = logger.log(
            level=LogLevel.ERROR,
            category=LogCategory.SYSTEM,
            message=f"Error clearing logs: {e!s}",
            details={"error": str(e)},
            http_status=500,
        )
        return jsonify(response), status
