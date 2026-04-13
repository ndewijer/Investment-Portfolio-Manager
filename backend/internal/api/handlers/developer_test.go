package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/handlers"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/testutil"
)

// newMultipartRequestWithFileContentType is like newMultipartRequest but lets the caller
// control the Content-Type header of the file part, to exercise validateCSVFile.
func newMultipartRequestWithFileContentType(t *testing.T, method, path string, fields map[string]string, filename, fileContentType, csvContent string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("WriteField %s: %v", k, err)
		}
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", fileContentType)
	fw, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := fw.Write([]byte(csvContent)); err != nil {
		t.Fatalf("Write csv: %v", err)
	}
	w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// newMultipartRequest builds a multipart/form-data request with an optional CSV file attachment.
func newMultipartRequest(t *testing.T, method, path string, fields map[string]string, filename, csvContent string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("WriteField %s: %v", k, err)
		}
	}
	if filename != "" {
		fw, err := w.CreateFormFile("file", filename)
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := fw.Write([]byte(csvContent)); err != nil {
			t.Fatalf("Write csv: %v", err)
		}
	}
	w.Close()
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func newDeveloperHandler(t *testing.T) *handlers.DeveloperHandler {
	t.Helper()
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestDeveloperService(t, db)
	return handlers.NewDeveloperHandler(svc)
}

// ---- GetLogs ----

//nolint:gocyclo // Comprehensive handler test with multiple subtests and assertions, cannot be split well.
func TestDeveloperHandler_GetLogs(t *testing.T) {
	t.Run("returns empty logs list", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs", nil)
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Logs == nil {
			t.Error("expected non-nil logs slice")
		}
		if len(resp.Logs) != 0 {
			t.Errorf("expected 0 logs, got %d", len(resp.Logs))
		}
	})

	t.Run("invalid perPage returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/logs",
			map[string]string{"perPage": "notanumber"})
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid startDate returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/logs",
			map[string]string{"startDate": "not-a-date"})
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns logs after delete seeds audit entry", func(t *testing.T) {
		handler := newDeveloperHandler(t)

		delReq := httptest.NewRequest(http.MethodDelete, "/api/developer/logs", nil)
		handler.DeleteLogs(httptest.NewRecorder(), delReq)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs", nil)
		w := httptest.NewRecorder()
		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var resp model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Logs) == 0 {
			t.Error("expected at least one log entry after delete")
		}
	})

	t.Run("level filter returns only matching logs", func(t *testing.T) {
		handler := newDeveloperHandler(t)

		// seed an INFO audit log via DeleteLogs
		handler.DeleteLogs(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/api/developer/logs", nil))

		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/logs",
			map[string]string{"level": "INFO"})
		w := httptest.NewRecorder()
		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var resp model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, l := range resp.Logs {
			if l.Level != "INFO" {
				t.Errorf("expected only INFO logs, got %s", l.Level)
			}
		}
	})

	t.Run("skip parameter passed through URL", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/logs",
			map[string]string{"skip": "200", "perPage": "50"})
		w := httptest.NewRecorder()

		handler.GetLogs(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp model.LogResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.HasMore {
			t.Error("expected has_more=false for skip beyond available logs")
		}
	})
}

// ---- GetLoggingConfig ----

func TestDeveloperHandler_GetLoggingConfig(t *testing.T) {
	t.Run("returns default config", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/system-settings/logging", nil)
		w := httptest.NewRecorder()

		handler.GetLoggingConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var setting model.LoggingSetting
		if err := json.NewDecoder(w.Body).Decode(&setting); err != nil {
			t.Fatalf("decode: %v", err)
		}
	})

	t.Run("reflects values set by SetLoggingConfig", func(t *testing.T) {
		handler := newDeveloperHandler(t)

		setReq := testutil.NewRequestWithBody(http.MethodPut, "/api/developer/system-settings/logging",
			`{"enabled": false, "level": "error"}`)
		handler.SetLoggingConfig(httptest.NewRecorder(), setReq)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/system-settings/logging", nil)
		w := httptest.NewRecorder()
		handler.GetLoggingConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		var setting model.LoggingSetting
		if err := json.NewDecoder(w.Body).Decode(&setting); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if setting.Enabled != false {
			t.Errorf("expected Enabled=false, got %v", setting.Enabled)
		}
		if setting.Level != "error" {
			t.Errorf("expected Level=error, got %s", setting.Level)
		}
	})
}

// ---- SetLoggingConfig ----

func TestDeveloperHandler_SetLoggingConfig(t *testing.T) {
	t.Run("valid enabled and level", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPut,
			"/api/developer/system-settings/logging",
			`{"enabled": true, "level": "info"}`)
		w := httptest.NewRecorder()

		handler.SetLoggingConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("missing enabled returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPut,
			"/api/developer/system-settings/logging",
			`{"level": "info"}`)
		w := httptest.NewRecorder()

		handler.SetLoggingConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid level returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPut,
			"/api/developer/system-settings/logging",
			`{"enabled": true, "level": "verbose"}`)
		w := httptest.NewRecorder()

		handler.SetLoggingConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPut,
			"/api/developer/system-settings/logging",
			`not json`)
		w := httptest.NewRecorder()

		handler.SetLoggingConfig(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("response body contains the values that were set", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPut,
			"/api/developer/system-settings/logging",
			`{"enabled": true, "level": "warning"}`)
		w := httptest.NewRecorder()

		handler.SetLoggingConfig(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var setting model.LoggingSetting
		if err := json.NewDecoder(w.Body).Decode(&setting); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if setting.Enabled != true {
			t.Errorf("expected Enabled=true, got %v", setting.Enabled)
		}
		if setting.Level != "warning" {
			t.Errorf("expected Level=warning, got %s", setting.Level)
		}
	})
}

// ---- GetFundPriceCSVTemplate ----

func TestDeveloperHandler_GetFundPriceCSVTemplate(t *testing.T) {
	t.Run("returns expected headers", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/csv/fund-prices/template", nil)
		w := httptest.NewRecorder()

		handler.GetFundPriceCSVTemplate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var tmpl model.TemplateModel
		if err := json.NewDecoder(w.Body).Decode(&tmpl); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tmpl.Headers) != 2 {
			t.Errorf("expected 2 headers, got %d", len(tmpl.Headers))
		}
	})
}

// ---- GetTransactionCSVTemplate ----

func TestDeveloperHandler_GetTransactionCSVTemplate(t *testing.T) {
	t.Run("returns expected headers", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/csv/transactions/template", nil)
		w := httptest.NewRecorder()

		handler.GetTransactionCSVTemplate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var tmpl model.TemplateModel
		if err := json.NewDecoder(w.Body).Decode(&tmpl); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(tmpl.Headers) != 4 {
			t.Errorf("expected 4 headers, got %d", len(tmpl.Headers))
		}
	})

	t.Run("description lists all four transaction types", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/csv/transactions/template", nil)
		w := httptest.NewRecorder()

		handler.GetTransactionCSVTemplate(w, req)

		var tmpl model.TemplateModel
		if err := json.NewDecoder(w.Body).Decode(&tmpl); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, typ := range []string{"buy", "sell", "dividend", "fee"} {
			if !strings.Contains(tmpl.Description, typ) {
				t.Errorf("expected description to mention %q", typ)
			}
		}
	})
}

// ---- GetExchangeRate ----

func TestDeveloperHandler_GetExchangeRate(t *testing.T) {
	t.Run("found returns rate", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		testutil.NewExchangeRate("USD", "EUR", "2024-01-15", 0.92).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/exchange-rate",
			map[string]string{
				"fromCurrency": "USD",
				"toCurrency":   "EUR",
				"date":         "2024-01-15",
			})
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp model.ExchangeRateWrapper
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Rate == nil {
			t.Fatal("expected rate, got nil")
		}
	})

	t.Run("not found returns null rate with 200", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/exchange-rate",
			map[string]string{
				"fromCurrency": "USD",
				"toCurrency":   "EUR",
				"date":         "2024-01-15",
			})
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp model.ExchangeRateWrapper
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Rate != nil {
			t.Errorf("expected nil rate, got %v", resp.Rate)
		}
	})

	t.Run("missing fromCurrency returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/exchange-rate",
			map[string]string{"toCurrency": "EUR", "date": "2024-01-15"})
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad date returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/exchange-rate",
			map[string]string{
				"fromCurrency": "USD",
				"toCurrency":   "EUR",
				"date":         "not-a-date",
			})
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing toCurrency returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/exchange-rate",
			map[string]string{"fromCurrency": "USD", "date": "2024-01-15"})
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing date returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/exchange-rate",
			map[string]string{"fromCurrency": "USD", "toCurrency": "EUR"})
		w := httptest.NewRecorder()

		handler.GetExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// ---- UpdateExchangeRate ----

func TestDeveloperHandler_UpdateExchangeRate(t *testing.T) {
	t.Run("upsert new rate", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate",
			`{"date":"2024-03-01","fromCurrency":"USD","toCurrency":"EUR","rate":"0.91"}`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("upsert existing rate", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		testutil.NewExchangeRate("USD", "EUR", "2024-03-01", 0.90).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate",
			`{"date":"2024-03-01","fromCurrency":"USD","toCurrency":"EUR","rate":"0.91"}`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid rate returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate",
			`{"date":"2024-03-01","fromCurrency":"USD","toCurrency":"EUR","rate":"notanumber"}`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate", `not json`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("negative rate returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate",
			`{"date":"2024-03-01","fromCurrency":"USD","toCurrency":"EUR","rate":"-0.5"}`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("zero rate returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate",
			`{"date":"2024-03-01","fromCurrency":"USD","toCurrency":"EUR","rate":"0"}`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing date returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/exchange-rate",
			`{"fromCurrency":"USD","toCurrency":"EUR","rate":"0.91"}`)
		w := httptest.NewRecorder()

		handler.UpdateExchangeRate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// ---- GetFundPrice ----

func TestDeveloperHandler_GetFundPrice(t *testing.T) {
	t.Run("found returns price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		priceDate, err := time.Parse("2006-01-02", "2024-06-01")
		if err != nil {
			t.Fatalf("parse date: %v", err)
		}
		testutil.NewFundPrice(fund.ID).WithDate(priceDate).WithPrice(100.0).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/fund-price",
			map[string]string{"fundId": fund.ID, "date": "2024-06-01"})
		w := httptest.NewRecorder()

		handler.GetFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/fund-price",
			map[string]string{"fundId": fund.ID, "date": "2024-06-01"})
		w := httptest.NewRecorder()

		handler.GetFundPrice(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("missing fundId returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/fund-price",
			map[string]string{"date": "2024-06-01"})
		w := httptest.NewRecorder()

		handler.GetFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad date returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithQueryParams(http.MethodGet, "/api/developer/fund-price",
			map[string]string{"fundId": testutil.MakeID(), "date": "not-a-date"})
		w := httptest.NewRecorder()

		handler.GetFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// ---- UpdateFundPrice ----

func TestDeveloperHandler_UpdateFundPrice(t *testing.T) {
	t.Run("upsert new price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		body, err := json.Marshal(map[string]string{
			"date":   "2024-03-01",
			"fundId": fund.ID,
			"price":  "123.45",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/fund-price", string(body))
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid price returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		body, err := json.Marshal(map[string]string{
			"date":   "2024-03-01",
			"fundId": fund.ID,
			"price":  "notanumber",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/fund-price", string(body))
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/fund-price", `not json`)
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("upsert existing price", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		testutil.NewFundPrice(fund.ID).WithDate(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)).WithPrice(100.0).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		body, err := json.Marshal(map[string]string{
			"date":   "2024-03-01",
			"fundId": fund.ID,
			"price":  "150.00",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/fund-price", string(body))
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("negative price returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		body, err := json.Marshal(map[string]string{
			"date":   "2024-03-01",
			"fundId": fund.ID,
			"price":  "-10.00",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/fund-price", string(body))
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("missing date returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		body, err := json.Marshal(map[string]string{
			"fundId": fund.ID,
			"price":  "100.00",
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		req := testutil.NewRequestWithBody(http.MethodPost, "/api/developer/fund-price", string(body))
		w := httptest.NewRecorder()

		handler.UpdateFundPrice(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// ---- DeleteLogs ----

func TestDeveloperHandler_DeleteLogs(t *testing.T) {
	t.Run("returns 204 no content", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodDelete, "/api/developer/logs", nil)
		w := httptest.NewRecorder()

		handler.DeleteLogs(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("logs cleared - only audit entry remains", func(t *testing.T) {
		handler := newDeveloperHandler(t)

		delReq := httptest.NewRequest(http.MethodDelete, "/api/developer/logs", nil)
		delW := httptest.NewRecorder()
		handler.DeleteLogs(delW, delReq)

		if delW.Code != http.StatusNoContent {
			t.Fatalf("delete failed: %d", delW.Code)
		}

		getReq := httptest.NewRequest(http.MethodGet, "/api/developer/logs", nil)
		getW := httptest.NewRecorder()
		handler.GetLogs(getW, getReq)

		var resp model.LogResponse
		if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Logs) != 1 {
			t.Errorf("expected 1 audit log after delete, got %d", len(resp.Logs))
		}
	})

	t.Run("X-Forwarded-For with multiple IPs stores only the first", func(t *testing.T) {
		handler := newDeveloperHandler(t)

		delReq := httptest.NewRequest(http.MethodDelete, "/api/developer/logs", nil)
		delReq.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.1, 172.16.0.1")
		handler.DeleteLogs(httptest.NewRecorder(), delReq)

		getReq := httptest.NewRequest(http.MethodGet, "/api/developer/logs", nil)
		getW := httptest.NewRecorder()
		handler.GetLogs(getW, getReq)

		var resp model.LogResponse
		if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Logs) == 0 {
			t.Fatal("expected audit log entry")
		}
		if resp.Logs[0].IPAddress != "1.2.3.4" {
			t.Errorf("expected IPAddress=1.2.3.4, got %q", resp.Logs[0].IPAddress)
		}
	})
}

// ---- ImportFundPrices ----

func TestDeveloperHandler_ImportFundPrices(t *testing.T) {
	t.Run("valid CSV imports rows", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		csv := "date,price\n2024-01-01,100.00\n2024-01-02,101.50\n"
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv", csv)
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]int
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result["imported"] != 2 {
			t.Errorf("expected 2 imported, got %d", result["imported"])
		}
	})

	t.Run("missing fundId returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			nil, "prices.csv", "date,price\n2024-01-01,100.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("fund not found returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": testutil.MakeID()}, "prices.csv", "date,price\n2024-01-01,100.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no file in form returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		// Pass empty filename so no file part is added to the multipart form
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "", "")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad file extension returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.txt", "date,price\n2024-01-01,100.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad content-type returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequestWithFileContentType(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv", "text/html", "date,price\n2024-01-01,100.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad headers returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv", "wrong,headers\n2024-01-01,100.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("BOM-prefixed CSV is handled correctly", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		bom := string([]byte{0xEF, 0xBB, 0xBF})
		csv := bom + "date,price\n2024-01-01,100.00\n"
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv", csv)
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for BOM CSV, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("empty CSV returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv", "")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty CSV, got %d", w.Code)
		}
	})

	t.Run("invalid date row returns 500", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv",
			"date,price\nnot-a-date,100.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("negative price row returns 500", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		fund := testutil.NewFund().Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-fund-prices",
			map[string]string{"fundId": fund.ID}, "prices.csv",
			"date,price\n2024-01-01,-5.00\n")
		w := httptest.NewRecorder()

		handler.ImportFundPrices(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

// ---- ImportTransactions ----

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestDeveloperHandler_ImportTransactions(t *testing.T) {
	t.Run("valid CSV imports rows", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		csv := "date,type,shares,cost_per_share\n2024-01-01,buy,10.0,100.00\n2024-01-02,sell,5.0,110.00\n"
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "transactions.csv", csv)
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]int
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result["imported"] != 2 {
			t.Errorf("expected 2 imported, got %d", result["imported"])
		}
	})

	t.Run("missing fundId returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			nil, "tx.csv", "date,type,shares,cost_per_share\n2024-01-01,buy,10,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("portfolio-fund not found returns 400", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": testutil.MakeID()}, "tx.csv",
			"date,type,shares,cost_per_share\n2024-01-01,buy,10,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("no file in form returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "", "")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad content-type returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequestWithFileContentType(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv", "text/html",
			"date,type,shares,cost_per_share\n2024-01-01,buy,10,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad headers returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv", "wrong,headers\n2024-01-01,buy\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("bad date row returns 500", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv",
			"date,type,shares,cost_per_share\nnot-a-date,buy,10,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("bad shares row returns 500", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv",
			"date,type,shares,cost_per_share\n2024-01-01,buy,notanumber,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("invalid type returns 500", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv",
			"date,type,shares,cost_per_share\n2024-01-01,transfer,10,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("empty CSV returns 400", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv", "")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("BOM-prefixed CSV is handled correctly", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		bom := "\xEF\xBB\xBF"
		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv",
			bom+"date,type,shares,cost_per_share\n2024-01-01,buy,10,100\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("dividend and fee transaction types are accepted", func(t *testing.T) {
		db := testutil.SetupTestDB(t)
		portfolio := testutil.NewPortfolio().Build(t, db)
		fund := testutil.NewFund().Build(t, db)
		pf := testutil.NewPortfolioFund(portfolio.ID, fund.ID).Build(t, db)
		svc := testutil.NewTestDeveloperService(t, db)
		handler := handlers.NewDeveloperHandler(svc)

		req := newMultipartRequest(t, http.MethodPost, "/api/developer/import-transactions",
			map[string]string{"fundId": pf.ID}, "tx.csv",
			"date,type,shares,cost_per_share\n2024-01-01,dividend,5,50\n2024-01-02,fee,1,10\n")
		w := httptest.NewRecorder()

		handler.ImportTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var result map[string]int
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if result["imported"] != 2 {
			t.Errorf("expected 2 imported rows, got %d", result["imported"])
		}
	})
}

// ---- GetLogFilterOptions ----

//nolint:gocyclo // Comprehensive integration test with multiple subtests
func TestDeveloperHandler_GetLogFilterOptions(t *testing.T) {
	t.Run("returns filter options successfully with empty database", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs/filter-options", nil)
		w := httptest.NewRecorder()

		handler.GetLogFilterOptions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var opts model.LogFilterOptions
		if err := json.NewDecoder(w.Body).Decode(&opts); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// All fields should be empty slices (no logs in DB)
		if len(opts.Levels) != 0 {
			t.Errorf("expected 0 levels, got %d", len(opts.Levels))
		}
		if len(opts.Categories) != 0 {
			t.Errorf("expected 0 categories, got %d", len(opts.Categories))
		}
		if len(opts.Sources) != 0 {
			t.Errorf("expected 0 sources, got %d", len(opts.Sources))
		}
		if len(opts.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(opts.Messages))
		}
	})

	t.Run("returns populated filter options after logs are created", func(t *testing.T) {
		handler := newDeveloperHandler(t)

		// Seed a log entry by calling DeleteLogs (creates an INFO-level audit log)
		delReq := httptest.NewRequest(http.MethodDelete, "/api/developer/logs", nil)
		handler.DeleteLogs(httptest.NewRecorder(), delReq)

		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs/filter-options", nil)
		w := httptest.NewRecorder()

		handler.GetLogFilterOptions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var opts model.LogFilterOptions
		if err := json.NewDecoder(w.Body).Decode(&opts); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(opts.Levels) == 0 {
			t.Error("expected at least one level after seeding logs")
		}
		if len(opts.Categories) == 0 {
			t.Error("expected at least one category after seeding logs")
		}
		if len(opts.Sources) == 0 {
			t.Error("expected at least one source after seeding logs")
		}
		if len(opts.Messages) == 0 {
			t.Error("expected at least one message after seeding logs")
		}
	})

	t.Run("response has correct JSON structure with all four fields", func(t *testing.T) {
		handler := newDeveloperHandler(t)
		req := httptest.NewRequest(http.MethodGet, "/api/developer/logs/filter-options", nil)
		w := httptest.NewRecorder()

		handler.GetLogFilterOptions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var raw map[string]json.RawMessage
		if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, field := range []string{"levels", "categories", "sources", "messages"} {
			if _, ok := raw[field]; !ok {
				t.Errorf("expected JSON field %q to be present in response", field)
			}
		}
		if len(raw) != 4 {
			t.Errorf("expected exactly 4 JSON fields, got %d", len(raw))
		}
	})
}
