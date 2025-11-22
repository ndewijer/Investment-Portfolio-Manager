import React, { useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { LoadingSpinner, ErrorMessage } from '../components/shared';
import {
  PortfolioSummary,
  PortfolioChart,
  FundsTable,
  TransactionsTable,
  DividendsTable,
  PortfolioActions,
} from '../components/portfolio';
import {
  usePortfolioData,
  useTransactionManagement,
  useDividendManagement,
} from '../hooks/portfolio';
import api from '../utils/api';
import './PortfolioDetail.css';

/**
 * Portfolio detail page
 *
 * Comprehensive view of a single portfolio including summary metrics, value chart,
 * fund holdings, transactions, and dividends. Orchestrates three custom hooks
 * (usePortfolioData, useTransactionManagement, useDividendManagement) to manage
 * separate data domains. Displays dividend table only if portfolio contains funds
 * with dividend_type !== 'none'.
 *
 * @returns {JSX.Element} The portfolio detail page
 */
const PortfolioDetail = () => {
  const { id } = useParams();

  // Portfolio data management
  const {
    portfolio,
    portfolioFunds,
    fundHistory,
    availableFunds,
    loading: portfolioLoading,
    error: portfolioError,
    fetchPortfolioData,
    loadAvailableFunds,
    refreshPortfolioSummary,
  } = usePortfolioData(id);

  // Transaction management
  const transactionState = useTransactionManagement(id, refreshPortfolioSummary);
  const {
    transactions,
    transactionsLoading,
    transactionsError,
    loadTransactions,
    handleEditTransaction,
    handleDeleteTransaction,
    openTransactionModal,
  } = transactionState;

  // Dividend management
  const dividendState = useDividendManagement(id, refreshPortfolioSummary);
  const {
    dividends,
    dividendsLoading,
    dividendsError,
    loadDividends,
    handleAddDividend,
    handleEditDividend,
    handleDeleteDividend,
  } = dividendState;

  // Load all data on component mount
  useEffect(() => {
    if (id) {
      fetchPortfolioData();
      loadTransactions();
      loadDividends();
    }
  }, [id, fetchPortfolioData, loadTransactions, loadDividends]);

  // Fund management functions
  const handleAddFund = useCallback(
    async (selectedFundId) => {
      try {
        await api.post('/portfolio-funds', {
          portfolio_id: id,
          fund_id: selectedFundId,
        });
        fetchPortfolioData();
      } catch (error) {
        console.error('Error adding fund to portfolio:', error);
        alert(error.response?.data?.user_message || 'Error adding fund to portfolio');
      }
    },
    [id, fetchPortfolioData]
  );

  const handleRemoveFund = useCallback(
    async (fund) => {
      try {
        await api.delete(`/portfolio-funds/${fund.id}`);
        fetchPortfolioData();
      } catch (error) {
        if (error.response && error.response.status === 409) {
          const data = error.response.data;
          const confirmMessage =
            `Are you sure you want to remove ${data.fund_name} from this portfolio?\n\n` +
            `This will also delete:\n` +
            `- ${data.transaction_count} transaction(s)\n` +
            `- ${data.dividend_count} dividend(s)\n\n` +
            `This action cannot be undone.`;

          if (window.confirm(confirmMessage)) {
            try {
              await api.delete(`/portfolio-funds/${fund.id}?confirm=true`);
              fetchPortfolioData();
              loadTransactions();
              loadDividends();
            } catch (confirmError) {
              console.error('Error removing fund after confirmation:', confirmError);
              alert(
                confirmError.response?.data?.user_message || 'Error removing fund from portfolio'
              );
            }
          }
        } else {
          console.error('Error removing fund:', error);
          alert(error.response?.data?.user_message || 'Error removing fund from portfolio');
        }
      }
    },
    [fetchPortfolioData, loadTransactions, loadDividends]
  );

  // Check for loading and error states
  const loading = portfolioLoading || transactionsLoading || dividendsLoading;
  const error = portfolioError || transactionsError || dividendsError;
  const hasDividendFunds = portfolioFunds.some((pf) => pf.dividend_type !== 'none');

  if (loading) return <LoadingSpinner message="Loading portfolio data..." />;
  if (error) return <ErrorMessage error={error} onRetry={fetchPortfolioData} showRetry={true} />;
  if (!portfolio) return <ErrorMessage error="Portfolio not found" />;

  return (
    <div className="portfolio-detail-container">
      <div className="portfolio-header">
        <h1>{portfolio.name}</h1>
        <p>{portfolio.description}</p>
      </div>

      <PortfolioSummary portfolio={portfolio} />

      <PortfolioChart fundHistory={fundHistory} portfolioFunds={portfolioFunds} />

      <FundsTable
        portfolioFunds={portfolioFunds}
        availableFunds={availableFunds}
        loading={portfolioLoading}
        error={portfolioError}
        onRetry={fetchPortfolioData}
        onAddFund={handleAddFund}
        onRemoveFund={handleRemoveFund}
        onAddTransaction={openTransactionModal}
        onAddDividend={handleAddDividend}
        onLoadAvailableFunds={loadAvailableFunds}
      />

      <TransactionsTable
        transactions={transactions}
        portfolioFunds={portfolioFunds}
        loading={transactionsLoading}
        error={transactionsError}
        onRetry={loadTransactions}
        onEditTransaction={handleEditTransaction}
        onDeleteTransaction={handleDeleteTransaction}
      />

      {hasDividendFunds && (
        <DividendsTable
          dividends={dividends}
          loading={dividendsLoading}
          error={dividendsError}
          onRetry={loadDividends}
          onEditDividend={handleEditDividend}
          onDeleteDividend={handleDeleteDividend}
        />
      )}

      <PortfolioActions
        transactionState={transactionState}
        dividendState={dividendState}
        portfolioFunds={portfolioFunds}
      />
    </div>
  );
};

export default PortfolioDetail;
