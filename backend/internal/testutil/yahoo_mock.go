package testutil

import (
	"context"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/yahoo"
)

// MockYahooClient is a mock implementation of the [yahoo.Client] interface for testing.
// It returns predefined test data instead of making actual API calls.
type MockYahooClient struct {
	// MockResponse is the response to return from query methods
	MockResponse yahoo.Response
	// MockError is the error to return from query methods
	MockError error
	// QueryCount tracks how many times a query method was called
	QueryCount int
}

// NewMockYahooClient creates a new mock Yahoo client with default test data.
// The default data includes 5 days of historical prices suitable for testing.
func NewMockYahooClient() *MockYahooClient {
	return &MockYahooClient{
		MockResponse: CreateMockYahooResponse(5),
		MockError:    nil,
		QueryCount:   0,
	}
}

// QueryYahooFiveDaySymbol mocks the 5-day symbol query with predefined test data.
// It returns the configured MockResponse and MockError.
func (m *MockYahooClient) QueryYahooFiveDaySymbol(_ context.Context, _ string) (yahoo.Response, error) {
	m.QueryCount++
	if m.MockError != nil {
		return yahoo.Response{}, m.MockError
	}
	return m.MockResponse, nil
}

// QueryYahooSymbolByDateRange mocks the date range query with predefined test data.
// It returns the configured MockResponse and MockError.
func (m *MockYahooClient) QueryYahooSymbolByDateRange(_ context.Context, _ string, _, _ time.Time) (yahoo.Response, error) {
	m.QueryCount++
	if m.MockError != nil {
		return yahoo.Response{}, m.MockError
	}
	return m.MockResponse, nil
}

// ParseChart delegates to the real ParseChart method since it's pure logic with no side effects.
func (m *MockYahooClient) ParseChart(yahooResult yahoo.Response) (yahoo.PriceChart, error) {
	// Use the real implementation for parsing since it's deterministic
	client := yahoo.NewFinanceClient()
	return client.ParseChart(yahooResult)
}

// WithError configures the mock to return the specified error.
func (m *MockYahooClient) WithError(err error) *MockYahooClient {
	m.MockError = err
	return m
}

// WithResponse configures the mock to return the specified response.
func (m *MockYahooClient) WithResponse(resp yahoo.Response) *MockYahooClient {
	m.MockResponse = resp
	return m
}

// WithEmptyResponse configures the mock to return an empty response (no data).
func (m *MockYahooClient) WithEmptyResponse() *MockYahooClient {
	m.MockResponse = yahoo.Response{
		Chart: yahoo.Chart{
			Result: []yahoo.Result{},
		},
	}
	return m
}

// CreateMockYahooResponse creates a mock Yahoo Finance API response with test data.
// The response includes `days` number of days of price data, ending yesterday.
// Each day has realistic OHLCV data suitable for testing.
func CreateMockYahooResponse(days int) yahoo.Response {
	now := time.Now().UTC()
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, time.UTC)

	timestamps := make([]int64, days)
	opens := make([]*float64, days)
	highs := make([]*float64, days)
	lows := make([]*float64, days)
	closes := make([]*float64, days)
	volumes := make([]*int64, days)

	// Generate realistic price data for testing
	basePrice := 100.0
	for i := range days {
		date := yesterday.AddDate(0, 0, -days+i+1)
		timestamps[i] = date.Unix()

		// Simulate price movement
		dayPrice := basePrice + float64(i)*0.5
		open := dayPrice
		high := dayPrice + 1.0
		low := dayPrice - 0.5
		closePrice := dayPrice + 0.25
		volume := int64(1000000 + i*10000)

		opens[i] = &open
		highs[i] = &high
		lows[i] = &low
		closes[i] = &closePrice
		volumes[i] = &volume
	}

	return yahoo.Response{
		Chart: yahoo.Chart{
			Result: []yahoo.Result{
				{
					Meta: yahoo.Meta{
						Symbol:           "TEST",
						Currency:         "USD",
						ExchangeName:     "NMS",
						FullExchangeName: "NASDAQ",
						LongName:         "Test Fund Inc.",
						Shortname:        "TEST",
					},
					Timestamp: timestamps,
					Indicators: yahoo.IndicatorsContainer{
						Quote: []yahoo.Quote{
							{
								Open:   opens,
								High:   highs,
								Low:    lows,
								Close:  closes,
								Volume: volumes,
							},
						},
					},
				},
			},
			Error: nil,
		},
	}
}

// CreateMockYahooResponseForDate creates a mock Yahoo response with a single day's data.
// Useful for testing specific date scenarios.
func CreateMockYahooResponseForDate(date time.Time, price float64) yahoo.Response {
	timestamp := date.Unix()
	volume := int64(1000000)

	return yahoo.Response{
		Chart: yahoo.Chart{
			Result: []yahoo.Result{
				{
					Meta: yahoo.Meta{
						Symbol:           "TEST",
						Currency:         "USD",
						ExchangeName:     "NMS",
						FullExchangeName: "NASDAQ",
						LongName:         "Test Fund Inc.",
						Shortname:        "TEST",
					},
					Timestamp: []int64{timestamp},
					Indicators: yahoo.IndicatorsContainer{
						Quote: []yahoo.Quote{
							{
								Open:   []*float64{&price},
								High:   []*float64{&price},
								Low:    []*float64{&price},
								Close:  []*float64{&price},
								Volume: []*int64{&volume},
							},
						},
					},
				},
			},
			Error: nil,
		},
	}
}

// CreateMockYahooErrorResponse creates a mock Yahoo response with an error.
// Useful for testing error handling scenarios.
func CreateMockYahooErrorResponse(errorMsg string) yahoo.Response {
	return yahoo.Response{
		Chart: yahoo.Chart{
			Result: []yahoo.Result{},
			Error:  &errorMsg,
		},
	}
}
