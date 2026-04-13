package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

var rglLog = logging.NewLogger("transaction")

// RealizedGainLossRepository provides data access methods for the realized_gain_loss table.
// It handles creating, retrieving, and deleting records of gains and losses from sold positions.
type RealizedGainLossRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewRealizedGainLossRepository creates a new RealizedGainLossRepository with the provided database connection.
func NewRealizedGainLossRepository(db *sql.DB) *RealizedGainLossRepository {
	return &RealizedGainLossRepository{db: db}
}

// WithTx returns a new RealizedGainLossRepository scoped to the provided transaction.
func (s *RealizedGainLossRepository) WithTx(tx *sql.Tx) *RealizedGainLossRepository {
	return &RealizedGainLossRepository{
		db: s.db,
		tx: tx,
	}
}

// getQuerier returns the active transaction if one is set, otherwise the database connection.
func (s *RealizedGainLossRepository) getQuerier() Querier {
	if s.tx != nil {
		return s.tx
	}
	return s.db
}

// GetRealizedGainLossByPortfolio retrieves all realized gain/loss records for the given portfolios within the specified date range.
// Records are filtered by transaction_date and sorted in ascending order by created_at.
//
// Parameters:
//   - portfolio: slice of portfolios to query
//   - startDate: inclusive start date for the query (compared against transaction_date)
//   - endDate: inclusive end date for the query (compared against transaction_date)
//
// Returns a map of portfolioID -> []RealizedGainLoss. If portfolio is empty, returns an empty map.
// Each record contains details about a sell transaction including shares sold, cost basis,
// sale proceeds, and the calculated realized gain or loss.
func (s *RealizedGainLossRepository) GetRealizedGainLossByPortfolio(portfolio []string, startDate, endDate time.Time) (map[string][]model.RealizedGainLoss, error) {
	rglLog.Debug("getting realized gain/loss by portfolio", "portfolio_count", len(portfolio), "start_date", startDate.Format("2006-01-02"), "end_date", endDate.Format("2006-01-02"))
	if len(portfolio) == 0 {
		return make(map[string][]model.RealizedGainLoss), nil
	}

	realizedGainLossPlaceholders := make([]string, len(portfolio))
	for i := range realizedGainLossPlaceholders {
		realizedGainLossPlaceholders[i] = "?"
	}

	//#nosec G202 -- Safe: placeholders are generated programmatically, not from user input
	realizedGainLossQuery := `
		SELECT id, portfolio_id, fund_id, transaction_id, transaction_date, shares_sold, cost_basis,
		sale_proceeds, realized_gain_loss, created_at
		FROM realized_gain_loss
		WHERE portfolio_id IN (` + strings.Join(realizedGainLossPlaceholders, ",") + `)
		AND transaction_date >= ?
		AND transaction_date <= ?
		ORDER BY transaction_date ASC, created_at ASC
	`

	realizedGainLossdArgs := make([]any, 0, len(portfolio)+2)
	for _, id := range portfolio {
		realizedGainLossdArgs = append(realizedGainLossdArgs, id)
	}
	realizedGainLossdArgs = append(realizedGainLossdArgs, startDate.Format("2006-01-02"))
	realizedGainLossdArgs = append(realizedGainLossdArgs, endDate.Format("2006-01-02"))

	rows, err := s.getQuerier().Query(realizedGainLossQuery, realizedGainLossdArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query realizedGainLoss table: %w", err)
	}
	defer rows.Close()

	realizedGainLosssByPortfolio := make(map[string][]model.RealizedGainLoss)

	for rows.Next() {
		var transactionDateStr, createdAtStr string
		var r model.RealizedGainLoss

		err := rows.Scan(
			&r.ID,
			&r.PortfolioID,
			&r.FundID,
			&r.TransactionID,
			&transactionDateStr,
			&r.SharesSold,
			&r.CostBasis,
			&r.SaleProceeds,
			&r.RealizedGainLoss,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan realizedGainLoss table results: %w", err)
		}

		r.TransactionDate, err = ParseTime(transactionDateStr)
		if err != nil || r.TransactionDate.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		r.CreatedAt, err = ParseTime(createdAtStr)
		if err != nil || r.CreatedAt.IsZero() {
			return nil, fmt.Errorf("failed to parse date: %w", err)
		}

		realizedGainLosssByPortfolio[r.PortfolioID] = append(realizedGainLosssByPortfolio[r.PortfolioID], r)

	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating realizedGainLoss table: %w", err)
	}

	return realizedGainLosssByPortfolio, nil
}

// InsertRealizedGainLoss creates a new realized gain/loss record in the database.
// All fields including ID must be set before calling this method.
func (s *RealizedGainLossRepository) InsertRealizedGainLoss(ctx context.Context, r *model.RealizedGainLoss) error {
	rglLog.DebugContext(ctx, "inserting realized gain/loss", "transaction_id", r.TransactionID, "portfolio_id", r.PortfolioID)
	query := `
		INSERT INTO realized_gain_loss (id, portfolio_id, fund_id, transaction_id, transaction_date,
			shares_sold, cost_basis, sale_proceeds, realized_gain_loss, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.getQuerier().ExecContext(ctx, query,
		r.ID,
		r.PortfolioID,
		r.FundID,
		r.TransactionID,
		r.TransactionDate.Format("2006-01-02"),
		r.SharesSold,
		r.CostBasis,
		r.SaleProceeds,
		r.RealizedGainLoss,
		r.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return fmt.Errorf("failed to insert realized gain/loss: %w", err)
	}

	return nil
}

// DeleteRealizedGainLossByTransactionID removes the realized gain/loss record associated with a transaction.
// Returns nil if no record exists for the given transaction ID (idempotent).
func (s *RealizedGainLossRepository) DeleteRealizedGainLossByTransactionID(ctx context.Context, transactionID string) error {
	rglLog.DebugContext(ctx, "deleting realized gain/loss by transaction ID", "transaction_id", transactionID)
	query := `DELETE FROM realized_gain_loss WHERE transaction_id = ?`

	_, err := s.getQuerier().ExecContext(ctx, query, transactionID)
	if err != nil {
		return fmt.Errorf("failed to delete realized gain/loss by transaction ID: %w", err)
	}

	return nil
}
