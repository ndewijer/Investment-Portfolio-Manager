package yahoo

import "time"

// Response represents the raw JSON response structure from Yahoo Finance API.
// This type maps directly to the Yahoo Finance chart API response format,
// containing nested structures for metadata, timestamps, and price indicators.
//
// The structure includes:
//   - Chart.Result: Array of result objects (typically contains one element)
//   - Chart.Result[].Meta: Symbol metadata (name, currency, exchange)
//   - Chart.Result[].Timestamp: Unix timestamps for each data point
//   - Chart.Result[].Indicators: Price data arrays (open, close, high, low, volume)
//   - Chart.Error: Optional error message from Yahoo API
type Response struct {
	Chart Chart `json:"chart"`
}

// Chart represents the chart container in the Yahoo Finance API response.
type Chart struct {
	Result []Result `json:"result"`
	Error  *string  `json:"error"`
}

// Result represents a single result item in the Yahoo Finance API response.
type Result struct {
	Meta       Meta                `json:"meta"`
	Timestamp  []int64             `json:"timestamp"`
	Indicators IndicatorsContainer `json:"indicators"`
}

// Meta represents symbol metadata in the Yahoo Finance API response.
type Meta struct {
	Currency         string `json:"currency"`
	Symbol           string `json:"symbol"`
	ExchangeName     string `json:"exchangeName"`
	FullExchangeName string `json:"fullExchangeName"`
	LongName         string `json:"longName"`
	Shortname        string `json:"shortName"`
}

// IndicatorsContainer holds the quote data arrays in the Yahoo Finance API response.
type IndicatorsContainer struct {
	Quote []Quote `json:"quote"`
}

// Quote represents the OHLCV price data arrays in the Yahoo Finance API response.
type Quote struct {
	Open   []*float64 `json:"open"`
	Close  []*float64 `json:"close"`
	Volume []*int64   `json:"volume"`
	High   []*float64 `json:"high"`
	Low    []*float64 `json:"low"`
}

// PriceChart represents a parsed and structured price chart from Yahoo Finance.
// This is the application's internal representation after parsing the raw Response.
//
// The chart contains:
//   - Symbol metadata: ticker, name, exchange, and currency information
//   - Indicators: A time-series array of daily price data points
//
// This structure is easier to work with than the raw Response and provides
// type-safe access to price data with proper time.Time dates.
type PriceChart struct {
	Currency         string       `json:"currency"`
	Symbol           string       `json:"symbol"`
	ExchangeName     string       `json:"exchangeName"`
	FullExchangeName string       `json:"fullExchangeName"`
	LongName         string       `json:"longName"`
	Shortname        string       `json:"shortName"`
	Indicators       []Indicators `json:"indicators"`
}

// Indicators represents a single day's price data for a financial instrument.
// Each Indicators instance corresponds to one trading day and contains the
// standard OHLCV (Open, High, Low, Close, Volume) data.
//
// Fields:
//   - Date: Trading date (time component set to midnight UTC)
//   - PriceOpen: Opening price for the day
//   - PriceClose: Closing price for the day
//   - PriceHigh: Highest price during the day
//   - PriceLow: Lowest price during the day
//   - Volume: Number of shares traded during the day
type Indicators struct {
	Date       time.Time
	PriceOpen  float64
	PriceClose float64
	Volume     int64
	PriceHigh  float64
	PriceLow   float64
}
