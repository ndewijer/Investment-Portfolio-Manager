package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
)

// NewRequestWithURLParams creates an HTTP request with chi URL parameters.
// This helper simplifies testing chi handlers that use chi.URLParam() to extract path parameters.
//
// Example:
//
//	req := testutil.NewRequestWithURLParams(
//	    http.MethodGet,
//	    "/api/portfolio/123-456",
//	    map[string]string{"portfolioId": "123-456"},
//	)
func NewRequestWithURLParams(method, path string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range params {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	return req
}

// NewRequestWithQueryParams creates an HTTP request with query parameters.
// This helper simplifies testing handlers that use r.URL.Query() to extract query string parameters.
//
// Example:
//
//	req := testutil.NewRequestWithQueryParams(
//	    http.MethodGet,
//	    "/api/portfolio/history",
//	    map[string]string{
//	        "start_date": "2024-01-01",
//	        "end_date": "2024-12-31",
//	    },
//	)
func NewRequestWithQueryParams(method, path string, queryParams map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	if len(queryParams) > 0 {
		q := req.URL.Query()
		for key, value := range queryParams {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req
}

// NewRequestWithQueryAndURLParams creates an HTTP request with both URL path parameters and query string parameters.
func NewRequestWithQueryAndURLParams(method, path string, URLparams map[string]string, queryParams map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	if len(URLparams) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range URLparams {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	if len(queryParams) > 0 {
		q := req.URL.Query()
		for key, value := range queryParams {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	return req
}

// NewRequestWithBody creates an HTTP request with a JSON body.
// This helper simplifies testing handlers that parse JSON request bodies.
//
// Example:
//
//	req := testutil.NewRequestWithBody(
//	    http.MethodPost,
//	    "/api/transaction",
//	    `{"portfolioFundId": "123", "shares": 100}`,
//	)
func NewRequestWithBody(method, path, body string) *http.Request {
	return httptest.NewRequest(method, path, strings.NewReader(body))
}

// NewRequestWithURLParamsAndBody creates an HTTP request with both chi URL parameters and a JSON body.
// This helper simplifies testing chi handlers that use chi.URLParam() and parse JSON request bodies.
//
// Example:
//
//	req := testutil.NewRequestWithURLParamsAndBody(
//	    http.MethodPut,
//	    "/api/transaction/123-456",
//	    map[string]string{"uuid": "123-456"},
//	    `{"shares": 150}`,
//	)
func NewRequestWithURLParamsAndBody(method, path string, params map[string]string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))

	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range params {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	return req
}
