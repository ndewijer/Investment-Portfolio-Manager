package testutil

import (
	"context"
	"sync"
	"time"
)

// InvalidatorCall records a single call to RegenerateMaterializedTable.
type InvalidatorCall struct {
	StartDate       time.Time
	PortfolioIDs    []string
	FundID          string
	PortfolioFundID string
}

// MockMaterializedInvalidator records calls to RegenerateMaterializedTable.
// It is safe for concurrent use from goroutines.
type MockMaterializedInvalidator struct {
	mu    sync.Mutex
	calls []InvalidatorCall
	// done is closed after each call so tests can wait for the async goroutine.
	done chan struct{}
}

// NewMockMaterializedInvalidator creates a new mock invalidator.
// expectedCalls sets the channel buffer size — set to the number of goroutine
// calls you expect so WaitForCall doesn't block indefinitely.
func NewMockMaterializedInvalidator(expectedCalls int) *MockMaterializedInvalidator {
	return &MockMaterializedInvalidator{
		done: make(chan struct{}, expectedCalls),
	}
}

// RegenerateMaterializedTable implements service.MaterializedInvalidator.
func (m *MockMaterializedInvalidator) RegenerateMaterializedTable(_ context.Context, startDate time.Time, portfolioIDs []string, fundID, portfolioFundID string) error {
	m.mu.Lock()
	m.calls = append(m.calls, InvalidatorCall{
		StartDate:       startDate,
		PortfolioIDs:    portfolioIDs,
		FundID:          fundID,
		PortfolioFundID: portfolioFundID,
	})
	m.mu.Unlock()

	// Signal that a call completed
	m.done <- struct{}{}
	return nil
}

// WaitForCall blocks until one call is recorded or the timeout expires.
// Returns true if a call was received, false on timeout.
func (m *MockMaterializedInvalidator) WaitForCall(timeout time.Duration) bool {
	select {
	case <-m.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Calls returns a snapshot of all recorded calls.
func (m *MockMaterializedInvalidator) Calls() []InvalidatorCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]InvalidatorCall, len(m.calls))
	copy(cp, m.calls)
	return cp
}

// CallCount returns the number of recorded calls.
func (m *MockMaterializedInvalidator) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}
