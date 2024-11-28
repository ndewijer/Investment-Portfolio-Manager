"""Add RealizedGainLoss table

Revision ID: 1.2.0
Revises: None
Create Date: 2024-03-21

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1.2.0'
down_revision = None
branch_labels = None
depends_on = None


def upgrade():
    # Create RealizedGainLoss table
    op.create_table(
        'realized_gain_loss',
        sa.Column('id', sa.String(36), primary_key=True),
        sa.Column('portfolio_id', sa.String(36), sa.ForeignKey('portfolio.id', ondelete='CASCADE'), nullable=False),
        sa.Column('fund_id', sa.String(36), sa.ForeignKey('fund.id'), nullable=False),
        sa.Column('transaction_date', sa.Date(), nullable=False),
        sa.Column('shares_sold', sa.Float(), nullable=False),
        sa.Column('cost_basis', sa.Float(), nullable=False),
        sa.Column('sale_proceeds', sa.Float(), nullable=False),
        sa.Column('realized_gain_loss', sa.Float(), nullable=False),
        sa.Column('created_at', sa.DateTime(), server_default=sa.text('CURRENT_TIMESTAMP')),
    )

    # Create indexes for better query performance
    op.create_index(
        'ix_realized_gain_loss_portfolio_id',
        'realized_gain_loss',
        ['portfolio_id']
    )
    op.create_index(
        'ix_realized_gain_loss_fund_id',
        'realized_gain_loss',
        ['fund_id']
    )
    op.create_index(
        'ix_realized_gain_loss_transaction_date',
        'realized_gain_loss',
        ['transaction_date']
    )


def downgrade():
    # Drop indexes first
    op.drop_index('ix_realized_gain_loss_transaction_date')
    op.drop_index('ix_realized_gain_loss_fund_id')
    op.drop_index('ix_realized_gain_loss_portfolio_id')
    
    # Drop the RealizedGainLoss table
    op.drop_table('realized_gain_loss')