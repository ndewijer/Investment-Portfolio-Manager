// Package ibkr provides an HTTP client for retrieving and parsing IBKR Flex Query reports.
package ibkr

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
)

var log = logging.NewLogger("ibkr")

// Client defines the interface for fetching financial data from Interactive Brokers.
// This interface enables dependency injection and testing with mock implementations.
type Client interface {
	RetreiveIbkrFlexReport(ctx context.Context, token string, queryID string) (FlexQueryResponse, []byte, error)
	TestIbkrConnection(ctx context.Context, token string, queryID string) (bool, error)
}

// FinanceClient provides methods for fetching financial data from Interactive Brokers.
// It wraps an HTTP client and provides convenient methods for querying flex queries.
//
// Concurrent calls to RetreiveIbkrFlexReport or TestIbkrConnection with the same queryID
// are deduplicated via an embedded singleflight.Group: only one in-flight HTTP round-trip
// is made to IBKR regardless of how many goroutines call simultaneously.
type FinanceClient struct {
	httpClient *http.Client
	sf         singleflight.Group // deduplicates concurrent IBKR API calls per queryID
}

// NewFinanceClient creates a new IBKR client with default HTTP settings.
// The client uses a standard http.Client for making requests to Interactive Brokers endpoints.
//
// Returns:
//   - *FinanceClient: A new client instance ready for use
func NewFinanceClient() *FinanceClient {
	return &FinanceClient{
		httpClient: &http.Client{},
	}
}

// flexReportResult carries the two return values of a Flex report fetch through
// singleflight's single interface{} return value, allowing waiters to unpack
// both the parsed response and the raw XML bytes from a shared in-flight call.
type flexReportResult struct {
	response FlexQueryResponse
	data     []byte
}

// RetreiveIbkrFlexReport fetches a Flex statement from the IBKR API.
// It first submits a request with the provided token and query ID to obtain a reference code,
// then polls until the statement is ready and returns the parsed response and raw XML bytes.
// Concurrent calls with the same queryID are collapsed into a single in-flight request via
// singleflight — only one call is made to IBKR regardless of how many callers are waiting.
// Returns an error if either the token or query ID is missing, or if the API call fails.
func (c *FinanceClient) RetreiveIbkrFlexReport(ctx context.Context, token string, queryID string) (FlexQueryResponse, []byte, error) {

	if token == "" || queryID == "" {
		return FlexQueryResponse{}, nil, fmt.Errorf("missing variables")
	}

	log.Debug("retrieving IBKR flex report", "query_id", queryID)

	v, err, _ := c.sf.Do(queryID, func() (any, error) {
		request, err := c.requestIBKRFlexReport(ctx, token, queryID)
		if err != nil {
			return nil, err
		}
		report, data, err := c.retrieveIBKRFlexReport(ctx, token, request)
		if err != nil {
			return nil, err
		}
		return flexReportResult{response: report, data: data}, nil
	})
	if err != nil {
		return FlexQueryResponse{}, nil, fmt.Errorf("retrieve IBKR flex report %s: %w", queryID, err)
	}

	r, ok := v.(flexReportResult)
	if !ok {
		return FlexQueryResponse{}, nil, fmt.Errorf("singleflight returned unexpected type")
	}

	log.Info("IBKR flex report retrieved", "query_id", queryID, "data_size", len(r.data))
	return r.response, r.data, nil
}

// TestIbkrConnection verifies that the provided token and queryID are accepted by IBKR.
// It submits a SendRequest call and returns true if IBKR responds without an error.
// Uses a "test:" key prefix on the singleflight group to avoid colliding with in-flight imports.
func (c *FinanceClient) TestIbkrConnection(ctx context.Context, token string, queryID string) (bool, error) {
	if token == "" || queryID == "" {
		return false, fmt.Errorf("missing variables")
	}

	log.Debug("testing IBKR connection", "query_id", queryID)

	_, err, _ := c.sf.Do("test:"+queryID, func() (any, error) {
		_, err := c.requestIBKRFlexReport(ctx, token, queryID)
		return nil, err
	})
	if err != nil {
		return false, fmt.Errorf("test IBKR connection %s: %w", queryID, err)
	}

	log.Info("IBKR connection test successful", "query_id", queryID)
	return true, nil
}

func (c *FinanceClient) requestIBKRFlexReport(ctx context.Context, token string, queryID string) (FlexRequestResponse, error) {
	if queryID == "" || token == "" {
		return FlexRequestResponse{}, fmt.Errorf("not all parameters set")
	}

	queryURL := fmt.Sprintf("https://ndcdyn.interactivebrokers.com/AccountManagement/FlexWebService/SendRequest?t=%s&q=%s&v=3", url.PathEscape(token), url.PathEscape(queryID))

	log.Debug("sending IBKR flex request", "query_id", queryID, "url", queryURL)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return FlexRequestResponse{}, fmt.Errorf("creating request: %w", err)
	}

	start := time.Now()

	//nolint:gosec // G704: host is hardcoded; token and queryID are DB-sourced query params, not URL components.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return FlexRequestResponse{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	elapsed := time.Since(start)
	if elapsed > 10*time.Second {
		log.Warn("slow IBKR SendRequest response", "elapsed", elapsed.String(), "query_id", queryID)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return FlexRequestResponse{}, fmt.Errorf("reading response body: %w", err)
	}

	log.Debug("IBKR SendRequest response received",
		"status_code", resp.StatusCode,
		"body_size", len(data),
		"elapsed", elapsed.String(),
	)

	var response FlexRequestResponse
	if err := xml.Unmarshal(data, &response); err != nil {
		return FlexRequestResponse{}, fmt.Errorf("parsing XML response: %w", err)
	}

	if response.ErrorCode != nil && response.ErrorMessage != nil {
		return response, fmt.Errorf("ibkr error %d: %s", *response.ErrorCode, *response.ErrorMessage)
	}

	log.Info("IBKR SendRequest successful", "query_id", queryID, "reference_code", response.ReferenceCode)
	return response, nil
}

//nolint:gocyclo // retry mechanism needs room to breath
func (c *FinanceClient) retrieveIBKRFlexReport(ctx context.Context, token string, request FlexRequestResponse) (FlexQueryResponse, []byte, error) {
	if request.Status == "fail" {
		return FlexQueryResponse{}, nil, fmt.Errorf("failed request submitted")
	}

	var response FlexQueryResponse
	var errResponse FlexRequestResponse
	var data []byte

	queryURL := fmt.Sprintf("%s?t=%s&q=%d&v=3", request.URL, token, request.ReferenceCode)

	log.Debug("retrieving IBKR flex report data", "reference_code", request.ReferenceCode, "url", queryURL)

	backoff := 2 * time.Second // start at 2s
	maxBackoff := 30 * time.Second
	maxAttempts := 10

	for attempt := range maxAttempts {
		if attempt > 0 {
			log.Warn("IBKR report not ready, retrying",
				"attempt", attempt+1,
				"max_attempts", maxAttempts,
				"backoff", backoff.String(),
				"reference_code", request.ReferenceCode,
			)
			select {
			case <-ctx.Done():
				return FlexQueryResponse{}, nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
		if err != nil {
			return FlexQueryResponse{}, nil, fmt.Errorf("creating request (attempt %d): %w", attempt+1, err)
		}

		start := time.Now()

		//nolint:gosec // G704: host is brought in from previous query; token and queryID are DB-sourced query params, not URL components.
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return FlexQueryResponse{}, nil, fmt.Errorf("executing request (attempt %d): %w", attempt+1, err)
		}

		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return FlexQueryResponse{}, nil, fmt.Errorf("reading response body (attempt %d): %w", attempt+1, err)
		}

		elapsed := time.Since(start)
		if elapsed > 10*time.Second {
			log.Warn("slow IBKR GetStatement response",
				"elapsed", elapsed.String(),
				"attempt", attempt+1,
				"reference_code", request.ReferenceCode,
			)
		}

		log.Debug("IBKR GetStatement response received",
			"attempt", attempt+1,
			"status_code", resp.StatusCode,
			"body_size", len(data),
			"elapsed", elapsed.String(),
		)

		errResponse = FlexRequestResponse{} // reset between attempts
		if err := xml.Unmarshal(data, &response); err != nil {
			if err := xml.Unmarshal(data, &errResponse); err != nil {
				return FlexQueryResponse{}, nil, fmt.Errorf("parsing XML response (attempt %d): %w", attempt+1, err)
			}
		}

		if errResponse.ErrorCode != nil &&
			(*errResponse.ErrorCode == 1018 || *errResponse.ErrorCode == 1019 || *errResponse.ErrorCode == 1021) {
			continue // not ready yet, retry
		}

		if errResponse.ErrorCode != nil {
			return FlexQueryResponse{}, nil, fmt.Errorf("ibkr error %d: %s", *errResponse.ErrorCode, *errResponse.ErrorMessage)
		}

		break // success
	}

	log.Info("IBKR flex report data retrieved", "reference_code", request.ReferenceCode, "data_size", len(data))
	response.ImportedAt = time.Now().UTC()
	response.QueryID = request.ReferenceCode
	return response, data, nil
}
