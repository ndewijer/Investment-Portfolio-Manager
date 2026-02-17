import React from 'react';
import { FormModal } from '../shared';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faMoneyBill, faChartLine, faCheck } from '@fortawesome/free-solid-svg-icons';
import NumericInput from '../NumericInput';
import { isDateInFuture } from '../../utils/portfolio/dateHelpers';

/**
 * PortfolioActions component - Modal collection for portfolio transactions and dividends
 *
 * Centralized component that renders all modal dialogs for portfolio actions:
 * - Add/Edit transaction modals (buy/sell)
 * - Add/Edit dividend modals (cash/stock)
 * - Handles reinvestment fields for stock dividends
 * - Auto-fetches historical prices for transaction dates
 * - Validates future dates for dividend reinvestments
 *
 * This component receives state management objects from the parent Portfolio
 * component and delegates rendering to FormModal components, keeping the parent
 * component clean and focused.
 *
 * @param {Object} props - Component props object
 * @param {Object} props.transactionState - Transaction state and handlers from parent
 * @param {Object} props.dividendState - Dividend state and handlers from parent
 * @param {Array} props.portfolioFunds - Portfolio funds data for display/validation
 * @returns {JSX.Element} Collection of modal dialogs for portfolio actions
 *
 * @example
 * <PortfolioActions
 *   transactionState={transactionState}
 *   dividendState={dividendState}
 *   portfolioFunds={portfolioFunds}
 * />
 */
const PortfolioActions = ({ transactionState, dividendState, portfolioFunds }) => {
  const {
    isTransactionModalOpen,
    isTransactionEditModalOpen,
    newTransaction,
    editingTransaction,
    priceFound,
    handleCreateTransaction,
    handleUpdateTransaction,
    handleTransactionDateChange,
    closeTransactionModal,
    closeEditModal,
    setNewTransaction,
    setEditingTransaction,
    setPriceFound,
  } = transactionState;

  const {
    isDividendModalOpen,
    isDividendEditModalOpen,
    newDividend,
    editingDividend,
    selectedFund,
    handleCreateDividend,
    handleUpdateDividend,
    closeDividendModal,
    closeDividendEditModal,
    setNewDividend,
    setEditingDividend,
  } = dividendState;

  return (
    <>
      {/* Transaction Modal */}
      <FormModal
        isOpen={isTransactionModalOpen}
        onClose={closeTransactionModal}
        title="Add Transaction"
        onSubmit={handleCreateTransaction}
      >
        <div className="form-group">
          <label>Fund:</label>
          <div className="static-field">
            {portfolioFunds.find((pf) => pf.id === newTransaction.portfolioFundId)?.fundName}
          </div>
        </div>
        <div className="form-group">
          <label>Date:</label>
          <input
            type="date"
            value={newTransaction.date}
            onChange={(e) => handleTransactionDateChange(e, portfolioFunds)}
            required
          />
        </div>
        <div className="form-group">
          <label>Type:</label>
          <select
            value={newTransaction.type}
            onChange={(e) =>
              setNewTransaction({
                ...newTransaction,
                type: e.target.value,
              })
            }
            required
          >
            <option value="buy">Buy</option>
            <option value="sell">Sell</option>
          </select>
        </div>
        <div className="form-group">
          <label>Shares:</label>
          <NumericInput
            value={newTransaction.shares}
            onChange={(value) =>
              setNewTransaction((prev) => ({
                ...prev,
                shares: value,
              }))
            }
            decimals={6}
            required
          />
        </div>
        <div className="form-group">
          <label>Cost per Share:</label>
          <div className="input-with-indicator">
            <NumericInput
              value={newTransaction.costPerShare}
              onChange={(value) => {
                setPriceFound(false);
                setNewTransaction((prev) => ({
                  ...prev,
                  costPerShare: value,
                }));
              }}
              decimals={2}
              required
            />
            {priceFound && <FontAwesomeIcon icon={faCheck} className="price-found-indicator" />}
          </div>
        </div>
      </FormModal>

      {/* Transaction Edit Modal */}
      <FormModal
        isOpen={isTransactionEditModalOpen}
        onClose={closeEditModal}
        title="Edit Transaction"
        onSubmit={handleUpdateTransaction}
      >
        {editingTransaction && (
          <>
            <div className="form-group">
              <label>Date:</label>
              <input
                type="date"
                value={editingTransaction.date}
                onChange={(e) =>
                  setEditingTransaction({
                    ...editingTransaction,
                    date: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Type:</label>
              <select
                value={editingTransaction.type}
                onChange={(e) =>
                  setEditingTransaction({
                    ...editingTransaction,
                    type: e.target.value,
                  })
                }
                required
              >
                <option value="buy">Buy</option>
                <option value="sell">Sell</option>
              </select>
            </div>
            <div className="form-group">
              <label>Shares:</label>
              <NumericInput
                value={editingTransaction.shares}
                onChange={(value) => {
                  setEditingTransaction((prev) => ({
                    ...prev,
                    shares: value,
                  }));
                }}
                decimals={6}
                required
              />
            </div>
            <div className="form-group">
              <label>Cost per Share:</label>
              <NumericInput
                value={editingTransaction.costPerShare}
                onChange={(value) => {
                  setEditingTransaction((prev) => ({
                    ...prev,
                    costPerShare: value,
                  }));
                }}
                decimals={2}
                required
              />
            </div>
          </>
        )}
      </FormModal>

      {/* Dividend Modal */}
      <FormModal
        isOpen={isDividendModalOpen}
        onClose={closeDividendModal}
        title="Add Dividend"
        onSubmit={handleCreateDividend}
      >
        <div className="form-group">
          <label>Fund:</label>
          <div className="static-field">
            {portfolioFunds.find((pf) => pf.id === newDividend.portfolioFundId)?.fundName}
          </div>
        </div>
        <div className="form-group">
          <label>Dividend Type:</label>
          <div className="static-field">
            {selectedFund?.dividendType === 'STOCK' ? (
              <>
                <FontAwesomeIcon icon={faChartLine} /> Stock Dividend
              </>
            ) : selectedFund?.dividendType === 'CASH' ? (
              <>
                <FontAwesomeIcon icon={faMoneyBill} /> Cash Dividend
              </>
            ) : (
              'No Dividend'
            )}
          </div>
        </div>
        <div className="form-group">
          <label>Record Date:</label>
          <input
            type="date"
            value={newDividend.recordDate}
            onChange={(e) =>
              setNewDividend({
                ...newDividend,
                recordDate: e.target.value,
              })
            }
            required
          />
        </div>
        <div className="form-group">
          <label>Ex-Dividend Date:</label>
          <input
            type="date"
            value={newDividend.exDividendDate}
            onChange={(e) =>
              setNewDividend({
                ...newDividend,
                exDividendDate: e.target.value,
              })
            }
            required
          />
        </div>
        <div className="form-group">
          <label>Dividend per Share:</label>
          <NumericInput
            value={newDividend.dividendPerShare}
            onChange={(value) => {
              setNewDividend((prev) => ({
                ...prev,
                dividendPerShare: value,
              }));
            }}
            decimals={2}
            required
          />
        </div>
        {selectedFund?.dividendType === 'STOCK' && (
          <div className="reinvestment-fields">
            <h3>
              Reinvestment Details{' '}
              {isDateInFuture(newDividend.exDividendDate) ? '(Optional)' : '(Required)'}
            </h3>
            <div className="form-group">
              <label>Buy Order Date:</label>
              <input
                type="date"
                value={newDividend.buyOrderDate || ''}
                onChange={(e) =>
                  setNewDividend({
                    ...newDividend,
                    buyOrderDate: e.target.value,
                  })
                }
              />
            </div>
            <div className="form-group">
              <label>Reinvestment Shares:</label>
              <NumericInput
                value={newDividend.reinvestmentShares}
                onChange={(value) => {
                  setNewDividend((prev) => ({
                    ...prev,
                    reinvestmentShares: value,
                  }));
                }}
                decimals={6}
              />
            </div>
            <div className="form-group">
              <label>Reinvestment Cost per Share:</label>
              <NumericInput
                value={newDividend.reinvestmentPrice}
                onChange={(value) => {
                  setNewDividend((prev) => ({
                    ...prev,
                    reinvestmentPrice: value,
                  }));
                }}
                decimals={2}
              />
            </div>
          </div>
        )}
      </FormModal>

      {/* Dividend Edit Modal */}
      <FormModal
        isOpen={isDividendEditModalOpen}
        onClose={closeDividendEditModal}
        title="Edit Dividend"
        onSubmit={handleUpdateDividend}
      >
        {editingDividend && (
          <>
            <div className="form-group">
              <label>Fund:</label>
              <div className="static-field">{editingDividend.fundName}</div>
            </div>
            <div className="form-group">
              <label>Record Date:</label>
              <input
                type="date"
                value={editingDividend.recordDate}
                onChange={(e) =>
                  setEditingDividend({
                    ...editingDividend,
                    recordDate: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Ex-Dividend Date:</label>
              <input
                type="date"
                value={editingDividend.exDividendDate}
                onChange={(e) =>
                  setEditingDividend({
                    ...editingDividend,
                    exDividendDate: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Dividend per Share:</label>
              <NumericInput
                value={editingDividend.dividendPerShare}
                onChange={(value) => {
                  setEditingDividend((prev) => ({
                    ...prev,
                    dividendPerShare: value,
                  }));
                }}
                decimals={2}
                required
              />
            </div>
            {selectedFund?.dividendType === 'STOCK' && (
              <div className="reinvestment-fields">
                <h3>
                  Reinvestment Details{' '}
                  {isDateInFuture(editingDividend.exDividendDate) ? '(Optional)' : '(Required)'}
                </h3>
                <div className="form-group">
                  <label>Buy Order Date:</label>
                  <input
                    type="date"
                    value={editingDividend.buyOrderDate || ''}
                    onChange={(e) =>
                      setEditingDividend({
                        ...editingDividend,
                        buyOrderDate: e.target.value,
                      })
                    }
                  />
                </div>
                <div className="form-group">
                  <label>Reinvestment Shares:</label>
                  <NumericInput
                    value={editingDividend.reinvestmentShares}
                    onChange={(value) => {
                      setEditingDividend((prev) => ({
                        ...prev,
                        reinvestmentShares: value,
                      }));
                    }}
                    decimals={6}
                  />
                </div>
                <div className="form-group">
                  <label>Reinvestment Price:</label>
                  <NumericInput
                    value={editingDividend.reinvestmentPrice}
                    onChange={(value) => {
                      setEditingDividend((prev) => ({
                        ...prev,
                        reinvestmentPrice: value,
                      }));
                    }}
                    decimals={2}
                  />
                </div>
              </div>
            )}
          </>
        )}
      </FormModal>
    </>
  );
};

export default PortfolioActions;
