import csv
from datetime import datetime
import uuid
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
        transactions = []
        required_fields = ['date', 'type', 'shares', 'cost_per_share']
        portfolio_fund = PortfolioFund.query.get(portfolio_fund_id)
        
        if not portfolio_fund:
            raise ValueError("Portfolio-fund relationship not found")

        try:
            decoded_content = file_content.decode('utf-8')
            csv_reader = csv.DictReader(decoded_content.splitlines())
            header_mapping = {}

            # Map headers to standardized names
            for std_name in required_fields:
                for actual_name in csv_reader.fieldnames:
                    if actual_name.lower().strip() == std_name:
                        header_mapping[actual_name] = std_name

            for row_num, row in enumerate(csv_reader, start=2):
                try:
                    # Map and sanitize fields
                    mapped_row = {}
                    for actual_name, value in row.items():
                        if actual_name in header_mapping:
                            mapped_row[header_mapping[actual_name]] = value

                    # Create transaction with UUID
                    transaction = Transaction(
                        id=str(uuid.uuid4()),  # Generate UUID for transaction
                        portfolio_fund_id=portfolio_fund.id,
                        date=DeveloperService.sanitize_date(mapped_row['date']),
                        type=DeveloperService.sanitize_string(mapped_row['type']).lower(),
                        shares=DeveloperService.sanitize_float(mapped_row['shares']),
                        cost_per_share=DeveloperService.sanitize_float(mapped_row['cost_per_share'])
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
        """Import fund prices from CSV file"""
        prices = []
        try:
            decoded_content = file_content.decode('utf-8')
            csv_reader = csv.DictReader(decoded_content.splitlines())

            for row_num, row in enumerate(csv_reader, start=2):
                try:
                    # Create fund price with UUID
                    fund_price = FundPrice(
                        id=str(uuid.uuid4()),  # Generate UUID for fund price
                        fund_id=fund_id,
                        date=DeveloperService.sanitize_date(row['date']),
                        price=DeveloperService.sanitize_float(row['price'])
                    )
                    prices.append(fund_price)

                except ValueError as e:
                    raise ValueError(f"Error in row {row_num}: {str(e)}")

            try:
                db.session.add_all(prices)
                db.session.commit()
                return len(prices)
            except Exception as e:
                db.session.rollback()
                raise ValueError(f"Database error: {str(e)}")

        except UnicodeDecodeError:
            raise ValueError("Invalid file encoding. Please use UTF-8 encoded CSV files.")
        except csv.Error as e:
            raise ValueError(f"CSV file error: {str(e)}")

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