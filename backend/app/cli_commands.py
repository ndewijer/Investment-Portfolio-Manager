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

    @app.cli.command("materialize-history")
    @click.option(
        "--portfolio-id",
        default=None,
        help="Specific portfolio ID to materialize (default: all portfolios)",
    )
    @click.option(
        "--force",
        is_flag=True,
        help="Force recalculation even if data already exists",
    )
    def materialize_history(portfolio_id, force):
        """
        Pre-calculate and store portfolio history for fast querying.

        This command calculates portfolio history values for all dates and stores
        them in the materialized view table. Future queries will use this cached
        data for much faster response times.

        Usage:
            flask materialize-history                    # Materialize all portfolios
            flask materialize-history --force            # Force recalculate all
            flask materialize-history --portfolio-id=abc # Specific portfolio
        """
        from .services.portfolio_history_materialized_service import (
            PortfolioHistoryMaterializedService,
        )

        click.echo("🔄 Starting portfolio history materialization...")
        click.echo("")

        if portfolio_id:
            # Materialize specific portfolio
            try:
                count = PortfolioHistoryMaterializedService.materialize_portfolio_history(
                    portfolio_id, force_recalculate=force
                )
                click.echo(f"✅ Materialized {count} records for portfolio {portfolio_id}")
            except ValueError as e:
                click.echo(f"❌ Error: {e}", err=True)
                return
        else:
            # Materialize all portfolios
            results = PortfolioHistoryMaterializedService.materialize_all_portfolios(
                force_recalculate=force
            )

            total_records = 0
            for pid, count in results.items():
                if isinstance(count, int):
                    click.echo(f"  ✅ Portfolio {pid}: {count} records")
                    total_records += count
                else:
                    click.echo(f"  ❌ Portfolio {pid}: {count}", err=True)

            click.echo("")
            click.echo(f"✅ Total records materialized: {total_records}")

        # Show stats
        stats = PortfolioHistoryMaterializedService.get_materialized_stats()
        click.echo("")
        click.echo("📊 Materialized View Statistics:")
        click.echo(f"  Total records: {stats['total_records']}")
        click.echo(f"  Portfolios with data: {stats['portfolios_with_data']}")
        if stats["oldest_date"]:
            click.echo(f"  Date range: {stats['oldest_date']} to {stats['newest_date']}")

    @app.cli.command("invalidate-materialized-history")
    @click.option(
        "--portfolio-id",
        required=True,
        help="Portfolio ID to invalidate",
    )
    @click.option(
        "--from-date",
        required=True,
        help="Invalidate from this date forward (YYYY-MM-DD)",
    )
    @click.option(
        "--recalculate",
        is_flag=True,
        help="Recalculate immediately after invalidation",
    )
    def invalidate_materialized_history(portfolio_id, from_date, recalculate):
        """
        Invalidate materialized history from a specific date forward.

        This is useful when you need to force recalculation due to data corrections
        or when automatic invalidation didn't trigger properly.

        Usage:
            flask invalidate-materialized-history --portfolio-id=abc --from-date=2024-01-01
            flask invalidate-materialized-history --portfolio-id=abc --from-date=2024-01-01 --recalculate
        """
        from datetime import datetime

        from .services.portfolio_history_materialized_service import (
            PortfolioHistoryMaterializedService,
        )

        try:
            from_date_parsed = datetime.strptime(from_date, "%Y-%m-%d").date()
        except ValueError:
            click.echo("❌ Invalid date format. Use YYYY-MM-DD", err=True)
            return

        click.echo(
            f"🔄 Invalidating materialized history for portfolio {portfolio_id} from {from_date}..."
        )

        try:
            deleted = PortfolioHistoryMaterializedService.invalidate_materialized_history(
                portfolio_id, from_date_parsed, recalculate=recalculate
            )
            click.echo(f"✅ Deleted {deleted} records")

            if recalculate:
                click.echo("✅ Recalculation completed")
        except Exception as e:
            click.echo(f"❌ Error: {e}", err=True)

    @app.cli.command("materialized-stats")
    def materialized_stats():
        """
        Show statistics about the materialized view.

        Usage:
            flask materialized-stats
        """
        from .services.portfolio_history_materialized_service import (
            PortfolioHistoryMaterializedService,
        )

        stats = PortfolioHistoryMaterializedService.get_materialized_stats()

        click.echo("📊 Materialized View Statistics:")
        click.echo(f"  Total records: {stats['total_records']}")
        click.echo(f"  Portfolios with data: {stats['portfolios_with_data']}")
        if stats["oldest_date"]:
            click.echo(f"  Date range: {stats['oldest_date']} to {stats['newest_date']}")
        else:
            click.echo("  No data in materialized view")

        if stats["total_records"] == 0:
            click.echo("")
            click.echo("💡 Run 'flask materialize-history' to populate the materialized view")
