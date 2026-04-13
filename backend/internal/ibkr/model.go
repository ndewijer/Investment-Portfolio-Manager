package ibkr

import (
	"encoding/xml"
	"time"
)

// FlexRequestResponse represents the initial response from the IBKR Flex Web Service
// when requesting a new statement. Contains the reference code used to retrieve the
// actual statement, or error details if the request failed.
type FlexRequestResponse struct {
	XMLName       xml.Name `xml:"FlexStatementResponse"`
	Text          string   `xml:",chardata"`
	Timestamp     string   `xml:"timestamp,attr"`
	Status        string   `xml:"Status"`        // Success or Fail
	ReferenceCode int      `xml:"ReferenceCode"` // Code to download the requested statement
	URL           string   `xml:"Url"`           // URL to download statement
	ErrorCode     *int     `xml:"ErrorCode"`     // If error, the error code
	ErrorMessage  *string  `xml:"ErrorMessage"`  // If error, the verbose message
}

// FlexQueryResponse represents the full Flex statement returned by IBKR after polling
// with a reference code. Contains trades, cash transactions, and conversion rates
// parsed from the XML response, plus metadata set after retrieval.
type FlexQueryResponse struct {
	XMLName        xml.Name `xml:"FlexQueryResponse"`
	Text           string   `xml:",chardata"`
	QueryName      string   `xml:"queryName,attr"`
	Type           string   `xml:"type,attr"`
	FlexStatements struct {
		Text          string `xml:",chardata"`
		Count         string `xml:"count,attr"`
		FlexStatement struct {
			Text          string `xml:",chardata"`
			AccountID     string `xml:"accountId,attr"`
			FromDate      string `xml:"fromDate,attr"`
			ToDate        string `xml:"toDate,attr"`
			Period        string `xml:"period,attr"`
			WhenGenerated string `xml:"whenGenerated,attr"`
			Trades        struct {
				Text  string `xml:",chardata"`
				Trade []struct {
					Text            string  `xml:",chardata"`
					Currency        string  `xml:"currency,attr"`
					CurrencyPrimary string  `xml:"currencyPrimary,attr"`
					Symbol          string  `xml:"symbol,attr"`
					Description     string  `xml:"description,attr"`
					Isin            string  `xml:"isin,attr"`
					Quantity        float64 `xml:"quantity,attr"`
					TradePrice      float64 `xml:"tradePrice,attr"`
					IbCommission    float64 `xml:"ibCommission,attr"`
					NetCash         float64 `xml:"netCash,attr"`
					IbOrderID       int64   `xml:"ibOrderID,attr"`
					TransactionID   int64   `xml:"transactionID,attr"`
					TradeDate       string  `xml:"tradeDate,attr"`
					Notes           string  `xml:"notes,attr"` // semicolon-separated; codes are case-sensitive ("Ri" != "RI")
					BuySell         string  `xml:"buySell,attr"`
					ReportDate      string  `xml:"reportDate,attr"`
				} `xml:"Trade"`
			} `xml:"Trades"`
			CashTransactions struct {
				Text            string `xml:",chardata"`
				CashTransaction []struct {
					Text          string  `xml:",chardata"`
					Currency      string  `xml:"currency,attr"`
					Symbol        string  `xml:"symbol,attr"`
					Description   string  `xml:"description,attr"`
					DateTime      string  `xml:"dateTime,attr"`
					Amount        float64 `xml:"amount,attr"`
					Type          string  `xml:"type,attr"`
					TransactionID int64   `xml:"transactionID,attr"`
					Isin          string  `xml:"isin,attr"`
					Code          string  `xml:"code,attr"`
					ReportDate    string  `xml:"reportDate,attr"`
					ExDate        string  `xml:"exDate,attr"`
				} `xml:"CashTransaction"`
			} `xml:"CashTransactions"`
			ConversionRates struct {
				Text           string `xml:",chardata"`
				ConversionRate []struct {
					Text         string  `xml:",chardata"`
					ReportDate   string  `xml:"reportDate,attr"`
					FromCurrency string  `xml:"fromCurrency,attr"`
					ToCurrency   string  `xml:"toCurrency,attr"`
					Rate         float64 `xml:"rate,attr"`
				} `xml:"ConversionRate"`
			} `xml:"ConversionRates"`
		} `xml:"FlexStatement"`
	} `xml:"FlexStatements"`
	ImportedAt time.Time
	QueryID    int
}
