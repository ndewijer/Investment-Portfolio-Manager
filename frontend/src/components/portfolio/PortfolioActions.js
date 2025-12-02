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
            {portfolioFunds.find((pf) => pf.id === newTransaction.portfolio_fund_id)?.fund_name}
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
              value={newTransaction.cost_per_share}
              onChange={(value) => {
                setPriceFound(false);
                setNewTransaction((prev) => ({
                  ...prev,
                  cost_per_share: value,
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
                value={editingTransaction.cost_per_share}
                onChange={(value) => {
                  setEditingTransaction((prev) => ({
                    ...prev,
                    cost_per_share: value,
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
            {portfolioFunds.find((pf) => pf.id === newDividend.portfolio_fund_id)?.fund_name}
          </div>
        </div>
        <div className="form-group">
          <label>Dividend Type:</label>
          <div className="static-field">
            {selectedFund?.dividend_type === 'stock' ? (
              <>
                <FontAwesomeIcon icon={faChartLine} /> Stock Dividend
              </>
            ) : selectedFund?.dividend_type === 'cash' ? (
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
            value={newDividend.record_date}
            onChange={(e) =>
              setNewDividend({
                ...newDividend,
                record_date: e.target.value,
              })
            }
            required
          />
        </div>
        <div className="form-group">
          <label>Ex-Dividend Date:</label>
          <input
            type="date"
            value={newDividend.ex_dividend_date}
            onChange={(e) =>
              setNewDividend({
                ...newDividend,
                ex_dividend_date: e.target.value,
              })
            }
            required
          />
        </div>
        <div className="form-group">
          <label>Dividend per Share:</label>
          <NumericInput
            value={newDividend.dividend_per_share}
            onChange={(value) => {
              setNewDividend((prev) => ({
                ...prev,
                dividend_per_share: value,
              }));
            }}
            decimals={2}
            required
          />
        </div>
        {selectedFund?.dividend_type === 'stock' && (
          <div className="reinvestment-fields">
            <h3>Reinvestment Details</h3>
            <div className="form-group">
              <label>Buy Order Date:</label>
              <input
                type="date"
                value={newDividend.buy_order_date || ''}
                onChange={(e) => {
                  const newDate = e.target.value;
                  const isFutureDate = isDateInFuture(newDate);

                  setNewDividend({
                    ...newDividend,
                    buy_order_date: newDate,
                    reinvestment_shares: isFutureDate ? '' : newDividend.reinvestment_shares,
                    reinvestment_price: isFutureDate ? '' : newDividend.reinvestment_price,
                  });
                }}
                required
              />
            </div>
            <div className="form-group">
              <label>Reinvestment Shares:</label>
              <NumericInput
                value={newDividend.reinvestment_shares}
                onChange={(value) => {
                  setNewDividend((prev) => ({
                    ...prev,
                    reinvestment_shares: value,
                  }));
                }}
                decimals={6}
                disabled={isDateInFuture(newDividend.buy_order_date)}
                required={!isDateInFuture(newDividend.buy_order_date)}
                className={isDateInFuture(newDividend.buy_order_date) ? 'disabled-input' : ''}
              />
            </div>
            <div className="form-group">
              <label>Reinvestment Cost per Share:</label>
              <NumericInput
                value={newDividend.reinvestment_price}
                onChange={(value) => {
                  setNewDividend((prev) => ({
                    ...prev,
                    reinvestment_price: value,
                  }));
                }}
                decimals={2}
                required
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
              <div className="static-field">{editingDividend.fund_name}</div>
            </div>
            <div className="form-group">
              <label>Record Date:</label>
              <input
                type="date"
                value={editingDividend.record_date}
                onChange={(e) =>
                  setEditingDividend({
                    ...editingDividend,
                    record_date: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Ex-Dividend Date:</label>
              <input
                type="date"
                value={editingDividend.ex_dividend_date}
                onChange={(e) =>
                  setEditingDividend({
                    ...editingDividend,
                    ex_dividend_date: e.target.value,
                  })
                }
                required
              />
            </div>
            <div className="form-group">
              <label>Dividend per Share:</label>
              <NumericInput
                value={editingDividend.dividend_per_share}
                onChange={(value) => {
                  setEditingDividend((prev) => ({
                    ...prev,
                    dividend_per_share: value,
                  }));
                }}
                decimals={2}
                required
              />
            </div>
            {selectedFund?.dividend_type === 'stock' && (
              <div className="reinvestment-fields">
                <h3>Reinvestment Details</h3>
                <div className="form-group">
                  <label>Buy Order Date:</label>
                  <input
                    type="date"
                    value={editingDividend.buy_order_date || ''}
                    onChange={(e) => {
                      const newDate = e.target.value;
                      setEditingDividend({
                        ...editingDividend,
                        buy_order_date: newDate,
                      });
                    }}
                    required
                  />
                </div>
                <div className="form-group">
                  <label>Reinvestment Shares:</label>
                  <NumericInput
                    value={editingDividend.reinvestment_shares}
                    onChange={(value) => {
                      setEditingDividend((prev) => ({
                        ...prev,
                        reinvestment_shares: value,
                      }));
                    }}
                    decimals={6}
                    disabled={isDateInFuture(editingDividend.buy_order_date)}
                    required={!isDateInFuture(editingDividend.buy_order_date)}
                    className={
                      isDateInFuture(editingDividend.buy_order_date) ? 'disabled-input' : ''
                    }
                  />
                </div>
                <div className="form-group">
                  <label>Reinvestment Price:</label>
                  <NumericInput
                    value={editingDividend.reinvestment_price}
                    onChange={(value) => {
                      setEditingDividend((prev) => ({
                        ...prev,
                        reinvestment_price: value,
                      }));
                    }}
                    decimals={2}
                    required
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
