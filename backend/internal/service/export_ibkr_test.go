package service

import (
	"context"

	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/model"
)

// ExportDecryptToken exposes decryptToken for testing.
func (s *IbkrService) ExportDecryptToken(token string) (string, error) {
	return s.decryptToken(token)
}

// ExportEncryptToken exposes encryptToken for testing.
func (s *IbkrService) ExportEncryptToken(token string) (string, error) {
	return s.encryptToken(token)
}

// ExportParseIBKRFlexReport exposes parseIBKRFlexReport for testing.
func (s *IbkrService) ExportParseIBKRFlexReport(report ibkr.FlexQueryResponse) ([]model.IBKRTransaction, []model.ExchangeRate, error) {
	return s.parseIBKRFlexReport(report)
}

// ExportWriteImportCache exposes writeImportCache for testing.
func (s *IbkrService) ExportWriteImportCache(ctx context.Context, cache model.IbkrImportCache) error {
	return s.writeImportCache(ctx, cache)
}

// ExportAddExchangeRates exposes addExchangeRates for testing.
func (s *IbkrService) ExportAddExchangeRates(ctx context.Context, rates []model.ExchangeRate) error {
	return s.addExchangeRates(ctx, rates)
}

// ExportFindFundByISINOrSymbol exposes findFundByISINOrSymbol for testing.
func (s *IbkrService) ExportFindFundByISINOrSymbol(isin, symbol string) (model.Fund, error) {
	return s.findFundByISINOrSymbol(isin, symbol)
}
