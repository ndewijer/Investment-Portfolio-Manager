package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fernet/fernet-go"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api"
	custommiddleware "github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/api/middleware"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/config"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/database"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/ibkr"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/logging"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/repository"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/service"
	"github.com/ndewijer/Investment-Portfolio-Manager/backend/internal/yahoo"
	"github.com/robfig/cron/v3"
)

var syslog = logging.NewLogger("system")

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := database.EnsureDir(cfg.Database.Path); err != nil {
		log.Fatalf("Failed to ensure database directory: %v", err)
	}
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize structured logging (reads config from DB if available).
	logHandler := logging.Init(db)
	defer logHandler.Close() // Flush remaining log entries before db.Close() (LIFO).

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	syslog.Info("connected to database", "path", cfg.Database.Path)

	// Resolve encryption key (env → file → auto-generate)
	dataDir := filepath.Dir(cfg.Database.Path)
	encKeyStr, err := resolveEncryptionKey(cfg.EncryptionKey, dataDir)
	if err != nil {
		log.Fatalf("Failed to resolve encryption key: %v", err)
	}

	var fernetKey *fernet.Key
	if encKeyStr != "" {
		fernetKey, err = fernet.DecodeKey(encKeyStr)
		if err != nil {
			log.Fatalf("Invalid IBKR_ENCRYPTION_KEY: %v", err)
		}
	}

	systemService, portfolioService, fundService, materializedService, dividendService, transactionService, ibkrService, developerService := createRepoAndServices(db, fernetKey)
	developerService.SetLogHandler(logHandler)

	// Create router
	router := api.NewRouter(
		systemService,
		portfolioService,
		fundService,
		materializedService,
		dividendService,
		transactionService,
		ibkrService,
		developerService,
		cfg,
	)

	// Build route -> handler function name lookup for request logging.
	custommiddleware.InitHandlerSources(router)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		syslog.Info("starting server", "addr", cfg.Server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	c := scheduleTasks(fundService, ibkrService)

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	syslog.Info("shutting down")

	// Stop accepting new HTTP requests + stop scheduling new cron jobs
	cronCtx := c.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Drain both in parallel under the same 30s window
	var shutdownErr bool
	if err := server.Shutdown(ctx); err != nil {
		syslog.Error("server shutdown error", "error", err)
		shutdownErr = true
	}

	select {
	case <-cronCtx.Done():
		syslog.Info("cron jobs finished cleanly")
	case <-ctx.Done():
		syslog.Warn("cron jobs did not finish in time")
		shutdownErr = true
	}

	if shutdownErr {
		log.Fatalf("Shutdown completed with errors")
	}
	syslog.Info("server exited")
}

func scheduleTasks(fundService *service.FundService, ibkrService *service.IbkrService) *cron.Cron {
	c := cron.New(
		cron.WithLocation(time.UTC),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
		),
	)
	// Schedule the price update task to run at 00:55 UTC every weekday
	_, err := c.AddFunc("55 00 * * 1-5", func() {
		syslog.Info("starting scheduled fund price update")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if _, err := fundService.UpdateAllFundHistory(ctx); err != nil {
			syslog.Error("scheduled fund price update failed", "error", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to register fund price update task: %v", err)
	}
	// Schedule the IBKR Import task to run between 05:30 and 07:30 UTC Tue-Sat
	// Fetches previous business day's close-of-business report
	_, err = c.AddFunc("30 5-7 * * 2-6", func() {
		syslog.Info("starting scheduled IBKR import")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if _, _, err := ibkrService.ImportFlexReport(ctx); err != nil {
			syslog.Error("scheduled IBKR import failed", "error", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to register IBKR import task: %v", err)
	}
	c.Start()
	return c
}

// resolveEncryptionKey returns the encryption key string from env, file, or auto-generation.
// Priority: env var > file > generate-and-write.
func resolveEncryptionKey(cfgKey, dataDir string) (string, error) {
	// 1. Env var already loaded into cfgKey
	if cfgKey != "" {
		return cfgKey, nil
	}

	// 2. Try file
	keyPath := filepath.Join(dataDir, ".ibkr_encryption_key")
	data, err := os.ReadFile(keyPath)
	if err == nil {
		key := strings.TrimSpace(string(data))
		if key != "" {
			return key, nil
		}
	}

	// 3. Generate new key and write to file
	var k fernet.Key
	if err := k.Generate(); err != nil {
		return "", fmt.Errorf("generate encryption key: %w", err)
	}
	encoded := k.Encode()

	if err := os.WriteFile(keyPath, []byte(encoded+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write encryption key to %s: %w", keyPath, err)
	}
	syslog.Info("generated new IBKR encryption key", "path", keyPath)

	return encoded, nil
}

//nolint:funlen // Wiring function that creates all repos and services; splitting would obscure the dependency graph.
func createRepoAndServices(db *sql.DB, fernetKey *fernet.Key) (
	*service.SystemService,
	*service.PortfolioService,
	*service.FundService,
	*service.MaterializedService,
	*service.DividendService,
	*service.TransactionService,
	*service.IbkrService,
	*service.DeveloperService,
) {
	// Create repositories
	portfolioRepo := repository.NewPortfolioRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	fundRepo := repository.NewFundRepository(db)
	pfRepo := repository.NewPortfolioFundRepository(db)
	dividendRepo := repository.NewDividendRepository(db)
	realizedGainLossRepo := repository.NewRealizedGainLossRepository(db)
	materializedRepo := repository.NewMaterializedRepository(db)
	ibkrRepo := repository.NewIbkrRepository(db)
	developerRepo := repository.NewDeveloperRepository(db)

	// Create services
	systemService := service.NewSystemService(db)

	// Create web clients
	yahooClient := yahoo.NewFinanceClient()
	ibkrClient := ibkr.NewFinanceClient()

	developerService := service.NewDeveloperService(
		db,
		developerRepo,
		fundRepo,
		transactionRepo,
		pfRepo,
	)

	realizedGainLossService := service.NewRealizedGainLossService(
		realizedGainLossRepo,
	)
	transactionService := service.NewTransactionService(
		db,
		transactionRepo,
		pfRepo,
		realizedGainLossRepo,
		ibkrRepo,
	)
	dividendService := service.NewDividendService(
		db,
		dividendRepo,
		pfRepo,
		transactionRepo,
	)
	portfolioService := service.NewPortfolioService(
		db,
		portfolioRepo,
		pfRepo,
	)
	dataloaderService := service.NewDataLoaderService(
		service.DataLoaderWithPortfolioFundRepository(pfRepo),
		service.DataLoaderWithFundRepository(fundRepo),
		service.DataLoaderWithTransactionService(transactionService),
		service.DataLoaderWithDividendService(dividendService),
		service.DataLoaderWithRealizedGainLossService(realizedGainLossService),
	)
	fundService := service.NewFundService(
		db,
		service.FundWithFundRepo(fundRepo),
		service.FundWithPortfolioFundRepo(pfRepo),
		service.FundWithTransactionService(transactionService),
		service.FundWithDividendService(dividendService),
		service.FundWithRealizedGainLossService(realizedGainLossService),
		service.FundWithDataLoaderService(dataloaderService),
		service.FundWithPortfolioRepo(portfolioRepo),
		service.FundWithYahooClient(yahooClient),
	)
	ibkrService := service.NewIbkrService(
		db,
		service.IbkrWithIbkrRepo(ibkrRepo),
		service.IbkrWithPortfolioRepo(portfolioRepo),
		service.IbkrWithFundRepo(fundRepo),
		service.IbkrWithDeveloperRepo(developerRepo),
		service.IbkrWithClient(ibkrClient),
		service.IbkrWithPortfolioFundRepo(pfRepo),
		service.IbkrWithTransactionRepo(transactionRepo),
		service.IbkrWithDividendRepo(dividendRepo),
		service.IbkrWithEncryptionKey(fernetKey),
	)
	materializedService := service.NewMaterializedService(db,
		service.MaterializedWithMaterializedRepository(materializedRepo),
		service.MaterializedWithPortfolioRepository(portfolioRepo),
		service.MaterializedWithFundRepository(fundRepo),
		service.MaterializedWithFundService(fundService),
		service.MaterializedWithDividendService(dividendService),
		service.MaterializedWithRealizedGainLossService(realizedGainLossService),
		service.MaterializedWithDataLoaderService(dataloaderService),
		service.MaterializedWithPortfolioService(portfolioService),
		service.MaterializedWithPortfolioFundRepository(pfRepo),
	)
	fundService.SetMaterializedInvalidator(materializedService)
	transactionService.SetMaterializedInvalidator(materializedService)
	dividendService.SetMaterializedInvalidator(materializedService)
	ibkrService.SetMaterializedInvalidator(materializedService)
	developerService.SetMaterializedInvalidator(materializedService)

	return systemService,
		portfolioService,
		fundService,
		materializedService,
		dividendService,
		transactionService,
		ibkrService,
		developerService
}
