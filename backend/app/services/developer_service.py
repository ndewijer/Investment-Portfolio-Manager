import csv
from datetime import datetime
from ..models import db, Transaction, PortfolioFund, FundPrice, Fund, Portfolio, ExchangeRate

class DeveloperService:
    @staticmethod
    def sanitize_string(value):
        """Sanitize string values"""
        if value is None:
            return None
        return str(value).strip()

    @staticmethod
    def sanitize_float(value):
        """Sanitize float values"""
        if value is None:
            return None
        try:
            return float(str(value).strip())
        except ValueError:
            raise ValueError(f"Invalid number format: {value}")

    @staticmethod
    def sanitize_date(value):
        """Sanitize date values"""
        if value is None:
            return None
        try:
            date_str = str(value).strip()
            return datetime.strptime(date_str, '%Y-%m-%d').date()
        except ValueError:
            raise ValueError(f"Invalid date format: {value}. Expected format: YYYY-MM-DD")

    @staticmethod
    def set_exchange_rate(from_currency, to_currency, rate, date=None):
        """Set exchange rate for a specific currency pair"""
        # Sanitize inputs
        from_currency = DeveloperService.sanitize_string(from_currency)
        to_currency = DeveloperService.sanitize_string(to_currency)
        rate = DeveloperService.sanitize_float(rate)
        date = DeveloperService.sanitize_date(date) or datetime.now().date()

        if not from_currency or not to_currency:
            raise ValueError("Currency codes cannot be empty")
        if not rate or rate <= 0:
            raise ValueError("Rate must be a positive number")
            
        exchange_rate = ExchangeRate.query.filter_by(
            from_currency=from_currency,
            to_currency=to_currency,
            date=date
        ).first()
        
        if exchange_rate:
            exchange_rate.rate = rate
        else:
            exchange_rate = ExchangeRate(
                from_currency=from_currency,
                to_currency=to_currency,
                rate=rate,
                date=date
            )
            db.session.add(exchange_rate)
            
        db.session.commit()
        return {
            'from_currency': exchange_rate.from_currency,
            'to_currency': exchange_rate.to_currency,
            'rate': exchange_rate.rate,
            'date': exchange_rate.date.isoformat()
        }

    @staticmethod
    def get_funds():
        """Get all available funds"""
        return Fund.query.all()

    @staticmethod
    def get_portfolios():
        """Get all available portfolios"""
        return Portfolio.query.all()

    @staticmethod
    def get_csv_template():
        """Get CSV template information"""
        return {
            'headers': ['date', 'type', 'shares', 'cost_per_share'],
            'example': {
                'date': '2024-03-21',
                'type': 'buy/sell',
                'shares': '10.5',
                'cost_per_share': '150.75'
            },
            'description': 'CSV file should contain the following columns:\n' +
                         '- date: Transaction date in YYYY-MM-DD format\n' +
                         '- type: Transaction type, either "buy" or "sell"\n' +
                         '- shares: Number of shares (decimal numbers allowed)\n' +
                         '- cost_per_share: Cost per share in the fund\'s currency'
        }

    @staticmethod
    def validate_utf8(file_content):
        """Validate that the file content is UTF-8 encoded"""
        try:
            # Try to decode as UTF-8 with strict error handling
            file_content.decode('utf-8', errors='strict')
            return True
        except UnicodeDecodeError as e:
            raise ValueError(f"File is not UTF-8 encoded. Please save the file in UTF-8 format. Error: {str(e)}")

    @staticmethod
    def import_transactions_csv(file_content, portfolio_fund_id):
        """Import transactions from CSV file"""
        try:
            # Validate UTF-8 encoding first and remove BOM if present
            content = file_content.decode('utf-8-sig')
            
            try:
                portfolio_fund_id = int(str(portfolio_fund_id).strip())
            except (ValueError, TypeError) as e:
                raise ValueError(f"Invalid portfolio_fund_id: {portfolio_fund_id}. Error: {str(e)}")

            # Get the portfolio_fund record with related portfolio and fund
            portfolio_fund = PortfolioFund.query.join(
                Portfolio, PortfolioFund.portfolio_id == Portfolio.id
            ).join(
                Fund, PortfolioFund.fund_id == Fund.id
            ).filter(
                PortfolioFund.id == portfolio_fund_id
            ).first()
            
            if not portfolio_fund:
                raise ValueError(f"No relationship found between portfolio and fund with ID {portfolio_fund_id}")
            
            if not portfolio_fund.portfolio or not portfolio_fund.fund:
                raise ValueError(
                    f"Invalid portfolio-fund relationship: Portfolio "
                    f"'{portfolio_fund.portfolio.name if portfolio_fund.portfolio else 'Unknown'}' "
                    f"and Fund '{portfolio_fund.fund.name if portfolio_fund.fund else 'Unknown'}'"
                )

            transactions = []
            # Split content into lines and create CSV reader
            lines = content.splitlines()
            if not lines:
                raise ValueError("CSV file is empty")
            
            csv_reader = csv.DictReader(lines)
            if not csv_reader.fieldnames:
                raise ValueError("CSV file has no headers")
            
            # Convert fieldnames to lowercase for case-insensitive comparison
            fieldnames_lower = [field.lower().strip() for field in csv_reader.fieldnames]
            
            # Verify all required fields are present in the header
            required_fields = {'date', 'type', 'shares', 'cost_per_share'}
            header_fields = set(fieldnames_lower)
            if not required_fields.issubset(header_fields):
                missing = required_fields - header_fields
                raise ValueError(f"Missing required columns in CSV header: {', '.join(missing)}\nFound headers: {', '.join(fieldnames_lower)}")
            
            # Create a mapping of actual header names to standardized names
            header_mapping = {}
            for std_name in required_fields:
                for actual_name in csv_reader.fieldnames:
                    if actual_name.lower().strip() == std_name:
                        header_mapping[actual_name] = std_name

            for row_num, row in enumerate(csv_reader, start=2):
                try:
                    # Map the headers to standardized names
                    mapped_row = {}
                    for actual_name, value in row.items():
                        if actual_name in header_mapping:
                            mapped_row[header_mapping[actual_name]] = value

                    # Sanitize each field
                    date = DeveloperService.sanitize_date(mapped_row['date'])
                    if not date:
                        raise ValueError(f"Invalid or missing date in row {row_num}")

                    transaction_type = DeveloperService.sanitize_string(mapped_row['type'])
                    if not transaction_type:
                        raise ValueError(f"Invalid or missing transaction type in row {row_num}")

                    # Validate transaction type
                    if transaction_type.lower() not in ['buy', 'sell']:
                        raise ValueError(f"Invalid transaction type in row {row_num}: {transaction_type}. Must be 'buy' or 'sell'")

                    try:
                        shares = DeveloperService.sanitize_float(mapped_row['shares'])
                        if shares is None or shares <= 0:
                            raise ValueError(f"Shares must be a positive number in row {row_num}")
                    except (ValueError, TypeError):
                        raise ValueError(f"Invalid shares value in row {row_num}: {mapped_row['shares']}")

                    try:
                        cost_per_share = DeveloperService.sanitize_float(mapped_row['cost_per_share'])
                        if cost_per_share is None or cost_per_share <= 0:
                            raise ValueError(f"Cost per share must be a positive number in row {row_num}")
                    except (ValueError, TypeError):
                        raise ValueError(f"Invalid cost per share value in row {row_num}: {mapped_row['cost_per_share']}")

                    transaction = Transaction(
                        portfolio_fund_id=portfolio_fund.id,
                        date=date,
                        type=transaction_type.lower(),
                        shares=shares,
                        cost_per_share=cost_per_share
                    )
                    transactions.append(transaction)

                except ValueError as e:
                    raise ValueError(f"Error in row {row_num}: {str(e)}")

            if not transactions:
                raise ValueError("No valid transactions found in CSV file")

            try:
                db.session.add_all(transactions)
                db.session.commit()
                return len(transactions)
            except Exception as e:
                db.session.rollback()
                raise ValueError(f"Database error while saving transactions: {str(e)}")

        except UnicodeDecodeError:
            raise ValueError("Invalid file encoding. Please use UTF-8 encoded CSV files.")
        except csv.Error as e:
            raise ValueError(f"CSV file error: {str(e)}")
        except Exception as e:
            if 'transactions' in locals():
                db.session.rollback()
            raise ValueError(f"Error processing CSV file: {str(e)}")

    @staticmethod
    def set_fund_price(fund_id, price, date=None):
        """Set the price for a fund on a specific date"""
        if date is None:
            date = datetime.now().date()
            
        fund_price = FundPrice(
            fund_id=fund_id,
            date=date,
            price=price
        )
        
        db.session.add(fund_price)
        db.session.commit()
        
        return {
            'fund_id': fund_id,
            'price': price,
            'date': date.isoformat()
        }

    @staticmethod
    def get_exchange_rate(from_currency, to_currency, date):
        """Get exchange rate for a specific currency pair and date"""
        exchange_rate = ExchangeRate.query.filter_by(
            from_currency=from_currency,
            to_currency=to_currency,
            date=date
        ).first()
        
        if exchange_rate:
            return {
                'from_currency': exchange_rate.from_currency,
                'to_currency': exchange_rate.to_currency,
                'rate': exchange_rate.rate,
                'date': exchange_rate.date.isoformat()
            }
        return None

    @staticmethod
    def import_fund_prices_csv(file_content, fund_id):
        """Import historical fund prices from CSV file"""
        try:
            # Validate UTF-8 encoding first
            DeveloperService.validate_utf8(file_content)
            
            fund_id = int(str(fund_id).strip())
        except ValueError as e:
            raise ValueError(str(e))

        # Verify the fund exists
        fund = Fund.query.get(fund_id)
        if not fund:
            raise ValueError(f"Fund {fund_id} not found")

        prices = []
        try:
            decoded_content = file_content.decode('utf-8').splitlines()
            csv_reader = csv.DictReader(decoded_content)
            
            for row_num, row in enumerate(csv_reader, start=2):  # start=2 because row 1 is headers
                # Check for missing required fields
                if 'date' not in row or 'price' not in row:
                    raise ValueError("CSV must contain 'date' and 'price' columns")
                
                if not row.get('date') or not row.get('price'):
                    raise ValueError(f"Missing value in row {row_num}: date and price are required")

                try:
                    # Sanitize each field
                    date = DeveloperService.sanitize_date(row['date'])
                    if not date:
                        raise ValueError(f"Invalid or missing date in row {row_num}")

                    price = DeveloperService.sanitize_float(row['price'])
                    if not price:
                        raise ValueError(f"Invalid or missing price in row {row_num}")

                    # Validate price
                    if price <= 0:
                        raise ValueError(f"Price must be positive in row {row_num}: {price}")

                    # Check for duplicate dates
                    existing_price = FundPrice.query.filter_by(
                        fund_id=fund_id,
                        date=date
                    ).first()

                    if existing_price:
                        # Update existing price
                        existing_price.price = price
                    else:
                        # Create new price
                        fund_price = FundPrice(
                            fund_id=fund_id,
                            date=date,
                            price=price
                        )
                        prices.append(fund_price)

                except ValueError as e:
                    raise ValueError(f"Error in row {row_num}: {str(e)}")

        except UnicodeDecodeError:
            raise ValueError("Invalid file encoding. Please use UTF-8 encoded CSV files.")
        except csv.Error as e:
            raise ValueError(f"CSV file error: {str(e)}")
        
        if not prices and not any(True for _ in csv_reader):
            raise ValueError("No valid data found in CSV file")

        try:
            db.session.add_all(prices)
            db.session.commit()
        except Exception as e:
            db.session.rollback()
            raise ValueError(f"Database error: {str(e)}")
        
        return len(prices)

    @staticmethod
    def get_fund_price_csv_template():
        """Get CSV template information for fund prices"""
        return {
            'headers': ['date', 'price'],
            'example': {
                'date': '2024-03-21',
                'price': '150.75'
            },
            'description': 'CSV file should contain the following columns:\n' +
                         '- date: Price date in YYYY-MM-DD format\n' +
                         '- price: Fund price (decimal numbers)'
        }