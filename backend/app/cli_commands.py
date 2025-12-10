"""CLI commands for managing seed data and maintenance tasks."""

import csv
import os
from datetime import datetime, timedelta

import click
import yfinance as yf
from flask import current_app


def register_commands(app):
    """Register CLI commands with the Flask app."""

    @app.cli.command("update-seed-prices")
    @click.option(
        "--days",
        default=365,
        help="Number of days of historical data to fetch (default: 365)",
    )
    def update_seed_prices(days):
        """
        Update seed fund price CSV files with latest data from yfinance.

        Fetches the latest price data for the seed funds and updates the CSV files
        in backend/data/seed/funds/prices/. This keeps the seed data current without
        requiring a rebuild.

        Usage:
            flask update-seed-prices
            flask update-seed-prices --days 730  # 2 years of data
        """
        # Seed funds with their Yahoo Finance tickers
        seed_funds = [
            {
                "isin": "US9229087690",
                "name": "Vanguard Total Stock Market ETF",
                "ticker": "VTI",
            },
            {
                "isin": "IE0003XJA0J9",
                "name": "Amundi Prime All Country World UCITS ETF Acc",
                "ticker": "WEBN.DE",
            },
            {
                "isin": "US0378331005",
                "name": "Apple Inc.",
                "ticker": "AAPL",
            },
        ]

        csv_dir = os.path.join(current_app.root_path, "..", "data", "seed", "funds", "prices")
        os.makedirs(csv_dir, exist_ok=True)

        end_date = datetime.now()
        start_date = end_date - timedelta(days=days)

        click.echo(f"Fetching {days} days of price data from yfinance...")
        click.echo(f"Date range: {start_date.date()} to {end_date.date()}")
        click.echo("")

        for fund in seed_funds:
            click.echo(f"Fetching {fund['name']} ({fund['ticker']})...")

            try:
                # Fetch data from yfinance
                ticker = yf.Ticker(fund["ticker"])
                hist = ticker.history(start=start_date, end=end_date)

                if hist.empty:
                    click.echo(f"  ❌ No data returned for {fund['ticker']}", err=True)
                    continue

                # Write to CSV
                csv_path = os.path.join(csv_dir, f"{fund['isin']}.csv")
                with open(csv_path, "w", newline="") as f:
                    writer = csv.writer(f)
                    writer.writerow(["date", "price"])

                    for date, row in hist.iterrows():
                        # Use closing price
                        price = row["Close"]
                        writer.writerow([date.strftime("%Y-%m-%d"), f"{price:.6f}"])

                rows_written = len(hist)
                click.echo(f"  ✅ Wrote {rows_written} rows to {fund['isin']}.csv")

            except Exception as e:
                click.echo(f"  ❌ Error fetching {fund['ticker']}: {e}", err=True)
                continue

        click.echo("")
        click.echo("✅ Seed price data updated successfully!")
        click.echo("Run 'flask seed-db' to regenerate seed data with the new prices.")
