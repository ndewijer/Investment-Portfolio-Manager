// Package yahoo provides an HTTP client for querying historical price data from Yahoo Finance.
package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
)

var log = logging.NewLogger("fund")

// Retry constants.
const (
	maxAttempts    = 3               // Total attempts per request.
	initialBackoff = 1 * time.Second // First retry wait.
	maxBackoff     = 4 * time.Second // Backoff cap.
)

// Client defines the interface for fetching financial data from Yahoo Finance API.
// This interface enables dependency injection and testing with mock implementations.
type Client interface {
	// QueryYahooFiveDaySymbol fetches the last 5 days of daily price data for a symbol.
	QueryYahooFiveDaySymbol(ctx context.Context, symbol string) (Response, error)

	// QueryYahooSymbolByDateRange fetches daily price data for a symbol within a date range.
	QueryYahooSymbolByDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) (Response, error)

	// ParseChart converts a raw Yahoo Finance API response into a structured price chart.
	ParseChart(yahooResult Response) (PriceChart, error)
}

// FinanceClient provides methods for fetching financial data from Yahoo Finance API.
// It wraps an HTTP client and provides convenient methods for querying stock prices
// and related financial data.
type FinanceClient struct {
	httpClient *http.Client
}

// NewFinanceClient creates a new Yahoo Finance client with default HTTP settings.
// The client uses a standard http.Client for making requests to Yahoo Finance endpoints.
//
// Returns:
//   - *FinanceClient: A new client instance ready for use
func NewFinanceClient() *FinanceClient {
	return &FinanceClient{
		httpClient: &http.Client{},
	}
}

// ParseChart converts a raw Yahoo Finance API response into a structured price chart.
// This method extracts price data (open, close, high, low, volume) and metadata
// (symbol, currency, exchange) from the Yahoo response format.
//
// The method performs validation to ensure:
//   - Timestamp data is present
//   - Close price data is present
//   - Data arrays have matching lengths
//
// Parameters:
//   - yahooResult: Raw response from Yahoo Finance API
//
// Returns:
//   - PriceChart: Structured chart with indicators and metadata
//   - error: If data is missing, malformed, or arrays have mismatched lengths
func (c *FinanceClient) ParseChart(yahooResult Response) (PriceChart, error) {
	log.Debug("parsing Yahoo chart response")
	if len(yahooResult.Chart.Result) == 0 {
		return PriceChart{}, fmt.Errorf("no results returned from Yahoo Finance")
	}

	result := yahooResult.Chart.Result[0]

	if len(result.Timestamp) == 0 {
		return PriceChart{}, fmt.Errorf("no price data returned")
	}
	if len(result.Indicators.Quote) == 0 || len(result.Indicators.Quote[0].Close) == 0 {
		return PriceChart{}, fmt.Errorf("no close prices returned")
	}

	if len(result.Indicators.Quote[0].Close) != len(result.Timestamp) {
		return PriceChart{}, fmt.Errorf("mismatched data lengths")
	}

	indicators := make([]Indicators, 0, len(result.Timestamp))

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i, v := range result.Timestamp {
		indicatorDate := time.Unix(v, 0).UTC().Truncate(24 * time.Hour)

		// Skip today's data - it's provisional intraday data that changes throughout the day.
		// Only use completed trading days (yesterday and older) to ensure data quality.
		if !indicatorDate.Before(today) {
			continue
		}

		if result.Indicators.Quote[0].Close[i] == nil {
			continue
		}

		indicator := Indicators{
			Date:       time.Unix(v, 0).UTC(),
			PriceClose: *result.Indicators.Quote[0].Close[i],
		}

		if result.Indicators.Quote[0].Open[i] != nil {
			indicator.PriceOpen = *result.Indicators.Quote[0].Open[i]
		}
		if result.Indicators.Quote[0].Volume[i] != nil {
			indicator.Volume = *result.Indicators.Quote[0].Volume[i]
		}
		if result.Indicators.Quote[0].High[i] != nil {
			indicator.PriceHigh = *result.Indicators.Quote[0].High[i]
		}
		if result.Indicators.Quote[0].Low[i] != nil {
			indicator.PriceLow = *result.Indicators.Quote[0].Low[i]
		}

		indicators = append(indicators, indicator)
	}

	log.Debug("parsed chart data",
		"symbol", result.Meta.Symbol,
		"currency", result.Meta.Currency,
		"indicator_count", len(indicators),
	)

	return PriceChart{
		Symbol:           result.Meta.Symbol,
		Currency:         result.Meta.Currency,
		ExchangeName:     result.Meta.ExchangeName,
		FullExchangeName: result.Meta.FullExchangeName,
		LongName:         result.Meta.LongName,
		Shortname:        result.Meta.Shortname,
		Indicators:       indicators,
	}, nil
}

// GetIndicatorForDate searches for price data matching a specific date.
// The method performs date-only comparison by truncating both the target and
// indicator dates to midnight UTC, ignoring time components.
//
// This is useful for finding a specific day's data when the exact timestamp
// is unknown or variable.
//
// Parameters:
//   - target: The date to search for (time component is ignored)
//
// Returns:
//   - Indicators: The price data for the matching date
//   - bool: true if a match was found, false otherwise
func (c PriceChart) GetIndicatorForDate(target time.Time) (Indicators, bool) {
	targetDay := target.UTC().Truncate(24 * time.Hour)
	for _, ind := range c.Indicators {
		if ind.Date.UTC().Truncate(24 * time.Hour).Equal(targetDay) {
			return ind, true
		}
	}
	return Indicators{}, false
}

// QueryYahooFiveDaySymbol fetches the last 5 days of daily price data for a symbol.
// This method is optimized for retrieving recent price history, typically used
// to get the latest available closing price.
//
// The method uses Yahoo Finance's range-based query format (range=5d) which
// automatically selects the most recent 5 trading days.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - symbol: Stock ticker symbol (e.g., "AAPL", "MSFT")
//
// Returns:
//   - Response: Raw API response containing price data
//   - error: If the HTTP request fails, API returns an error, or no results found
func (c *FinanceClient) QueryYahooFiveDaySymbol(ctx context.Context, symbol string) (Response, error) {
	queryURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=5d", url.PathEscape(symbol))

	log.DebugContext(ctx, "querying yahoo 5-day symbol", "symbol", symbol, "url", queryURL)

	result, err := c.queryYahoo(ctx, queryURL)
	if err != nil {
		return Response{}, fmt.Errorf("yahoo 5-day query for %s: %w", symbol, err)
	}
	if len(result.Chart.Result) == 0 {
		return Response{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	log.InfoContext(ctx, "yahoo 5-day query successful", "symbol", symbol, "result_count", len(result.Chart.Result))
	return result, nil
}

// QueryYahooSymbolByDateRange fetches daily price data for a symbol within a specific date range.
// This method is optimized for historical data backfilling, allowing retrieval of
// price data for any arbitrary date range.
//
// The method uses Yahoo Finance's period-based query format with Unix timestamps,
// providing precise control over the requested date range.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - symbol: Stock ticker symbol (e.g., "AAPL", "MSFT")
//   - startDate: Beginning of date range (inclusive)
//   - endDate: End of date range (inclusive)
//
// Returns:
//   - Response: Raw API response containing price data for the range
//   - error: If the HTTP request fails, API returns an error, or no results found
func (c *FinanceClient) QueryYahooSymbolByDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) (Response, error) {
	// Yahoo's period2 parameter is exclusive, so add 1 day to endDate to ensure
	// we get all data up to and including the endDate
	adjustedEndDate := endDate.AddDate(0, 0, 1)

	queryURL := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d",
		url.PathEscape(symbol),
		startDate.Unix(),
		adjustedEndDate.Unix(),
	)

	log.DebugContext(ctx, "querying yahoo by date range",
		"symbol", symbol,
		"start_date", startDate.Format("2006-01-02"),
		"end_date", endDate.Format("2006-01-02"),
		"url", queryURL,
	)

	result, err := c.queryYahoo(ctx, queryURL)
	if err != nil {
		return Response{}, fmt.Errorf("yahoo date-range query for %s (%s to %s): %w",
			symbol, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), err)
	}
	if len(result.Chart.Result) == 0 {
		return Response{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	log.InfoContext(ctx, "yahoo date-range query successful",
		"symbol", symbol,
		"start_date", startDate.Format("2006-01-02"),
		"end_date", endDate.Format("2006-01-02"),
		"result_count", len(result.Chart.Result),
	)
	return result, nil
}

// isRetryableStatus returns true for HTTP status codes that warrant a retry.
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code <= 599)
}

// queryYahoo executes an HTTP request to Yahoo Finance API with retry logic.
// Retries up to 3 times with exponential backoff (1s → 2s → 4s) on transient
// failures: HTTP 5xx, 429 (rate limit), and network/timeout errors.
// Permanent failures (4xx except 429, JSON parse errors, Yahoo API errors) fail immediately.
//
//nolint:gocyclo // Retry loop with error classification needs the branches.
func (c *FinanceClient) queryYahoo(ctx context.Context, queryURL string) (Response, error) {
	backoff := initialBackoff
	var lastErr error

	for attempt := range maxAttempts {
		if attempt > 0 {
			log.WarnContext(ctx, "yahoo API request failed, retrying",
				"attempt", attempt+1,
				"max_attempts", maxAttempts,
				"backoff", backoff.String(),
				"error", lastErr.Error(),
				"url", queryURL,
			)
			select {
			case <-ctx.Done():
				return Response{}, fmt.Errorf("context cancelled waiting for retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
		if err != nil {
			return Response{}, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Accept", "application/json")

		start := time.Now()

		//nolint:gosec // G704: URL is constructed from a hardcoded Yahoo Finance base with url.PathEscape applied to the symbol; the host cannot be redirected by user input.
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Network/timeout errors are retryable.
			lastErr = fmt.Errorf("executing request (attempt %d): %w", attempt+1, err)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body (attempt %d): %w", attempt+1, err)
			continue
		}

		elapsed := time.Since(start)
		if elapsed > 5*time.Second {
			log.WarnContext(ctx, "slow yahoo API response",
				"elapsed", elapsed.String(),
				"attempt", attempt+1,
				"url", queryURL,
			)
		}

		log.DebugContext(ctx, "yahoo API response received",
			"attempt", attempt+1,
			"status_code", resp.StatusCode,
			"body_size", len(data),
			"elapsed", elapsed.String(),
		)

		// Retry on transient HTTP errors (5xx, 429).
		if isRetryableStatus(resp.StatusCode) {
			lastErr = fmt.Errorf("HTTP %d (attempt %d)", resp.StatusCode, attempt+1)
			continue
		}

		// Non-retryable HTTP error — fail immediately.
		if resp.StatusCode != http.StatusOK {
			return Response{}, fmt.Errorf("HTTP %d from yahoo API", resp.StatusCode)
		}

		var response Response
		if err := json.Unmarshal(data, &response); err != nil {
			// JSON parse failure is not retryable — the server returned garbage or a non-JSON page.
			return Response{}, fmt.Errorf("parsing JSON response: %w", err)
		}

		if response.Chart.Error != nil {
			// Yahoo API-level errors are not retryable (bad symbol, invalid range, etc.).
			return response, fmt.Errorf("yahoo error: %s", *response.Chart.Error)
		}

		return response, nil
	}

	// All attempts exhausted.
	return Response{}, fmt.Errorf("yahoo API request failed after %d attempts: %w", maxAttempts, lastErr)
}
