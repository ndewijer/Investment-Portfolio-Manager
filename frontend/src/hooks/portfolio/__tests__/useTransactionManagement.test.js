/**
 * @fileoverview Test suite for useTransactionManagement custom hook
 *
 * Tests transaction CRUD operations and state management:
 * - Modal state management (open/close for create and edit)
 * - Create transaction with API integration
 * - Edit transaction workflow
 * - Update transaction with API integration
 * - Delete transaction with confirmation
 * - Date change with automatic price lookup
 * - Transaction form state management
 *
 * Testing approach:
 * - Mock useApiState hook for transaction data management
 * - Mock API calls (api.post, api.put, api.delete, api.get)
 * - Mock window.confirm for delete confirmation
 * - Use renderHook and act from @testing-library/react
 * - Target coverage: 70%+ focusing on critical business logic
 *
 * Total: 25+ tests
 */
import { renderHook, act } from '@testing-library/react';
import { useTransactionManagement } from '../useTransactionManagement';
import useApiState from '../../useApiState';
import api from '../../../utils/api';
import { getTodayString } from '../../../utils/portfolio/dateHelpers';

// Mock dependencies
jest.mock('../../useApiState');
jest.mock('../../../utils/api');

describe('useTransactionManagement', () => {
  let mockExecute;
  let mockOnDataChange;
  const mockPortfolioId = 'test-portfolio-id';

  beforeEach(() => {
    // Reset mocks
    jest.clearAllMocks();

    // Mock useApiState
    mockExecute = jest.fn();
    useApiState.mockReturnValue({
      data: [],
      loading: false,
      error: null,
      execute: mockExecute,
    });

    // Mock onDataChange callback
    mockOnDataChange = jest.fn();

    // Mock window.confirm
    global.window.confirm = jest.fn(() => true);

    // Mock console methods
    global.console.error = jest.fn();
    global.alert = jest.fn();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Initialization', () => {
    test('initializes with correct default state', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      expect(result.current.transactions).toEqual([]);
      expect(result.current.transactionsLoading).toBe(false);
      expect(result.current.transactionsError).toBeNull();
      expect(result.current.isTransactionModalOpen).toBe(false);
      expect(result.current.isTransactionEditModalOpen).toBe(false);
      expect(result.current.editingTransaction).toBeNull();
      expect(result.current.priceFound).toBe(false);
    });

    test('initializes newTransaction with correct defaults', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      expect(result.current.newTransaction).toMatchObject({
        portfolioFundId: '',
        date: getTodayString(),
        type: 'buy',
        shares: '',
        costPerShare: '',
      });
    });
  });

  describe('Modal Management', () => {
    test('openTransactionModal sets fund ID and opens modal', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.openTransactionModal('fund-123');
      });

      expect(result.current.isTransactionModalOpen).toBe(true);
      expect(result.current.newTransaction.portfolioFundId).toBe('fund-123');
      expect(result.current.newTransaction.date).toBe(getTodayString());
      expect(result.current.newTransaction.type).toBe('buy');
    });

    test('closeTransactionModal resets form and closes modal', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      // Open modal first
      act(() => {
        result.current.openTransactionModal('fund-123');
      });

      // Then close
      act(() => {
        result.current.closeTransactionModal();
      });

      expect(result.current.isTransactionModalOpen).toBe(false);
      expect(result.current.newTransaction.portfolioFundId).toBe('');
      expect(result.current.newTransaction.shares).toBe('');
      expect(result.current.newTransaction.costPerShare).toBe('');
    });

    test('handleEditTransaction opens edit modal with transaction data', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      const mockTransaction = {
        id: 1,
        portfolioFundId: 'fund-123',
        date: '2024-01-15T00:00:00Z',
        type: 'buy',
        shares: 100,
        costPerShare: 50,
      };

      act(() => {
        result.current.handleEditTransaction(mockTransaction);
      });

      expect(result.current.isTransactionEditModalOpen).toBe(true);
      expect(result.current.editingTransaction).toMatchObject({
        id: 1,
        portfolioFundId: 'fund-123',
        date: '2024-01-15',
        type: 'buy',
        shares: 100,
        costPerShare: 50,
      });
    });

    test('closeEditModal closes edit modal and clears editing state', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      // Open edit modal first
      act(() => {
        result.current.handleEditTransaction({
          id: 1,
          date: '2024-01-15',
          portfolioFundId: 'fund-123',
        });
      });

      // Then close
      act(() => {
        result.current.closeEditModal();
      });

      expect(result.current.isTransactionEditModalOpen).toBe(false);
      expect(result.current.editingTransaction).toBeNull();
    });
  });

  describe('Load Transactions', () => {
    test('loadTransactions fetches transactions for portfolio', async () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      await act(async () => {
        await result.current.loadTransactions();
      });

      // Verify that execute was called with a function that would call the API
      expect(mockExecute).toHaveBeenCalled();
      const executedFunction = mockExecute.mock.calls[0][0];
      expect(typeof executedFunction).toBe('function');
    });

    test('loadTransactions does nothing if no portfolioId', async () => {
      const { result } = renderHook(() => useTransactionManagement(null, mockOnDataChange));

      await act(async () => {
        await result.current.loadTransactions();
      });

      expect(mockExecute).not.toHaveBeenCalled();
    });
  });

  describe('Create Transaction', () => {
    test('handleCreateTransaction creates transaction and updates state', async () => {
      const mockNewTransaction = {
        portfolioFundId: 'fund-123',
        date: '2024-01-15',
        type: 'buy',
        shares: 100,
        costPerShare: 50,
      };

      const mockCreatedTransaction = { ...mockNewTransaction, id: 1 };

      api.post.mockResolvedValue({ data: mockCreatedTransaction });

      useApiState.mockReturnValue({
        data: [],
        loading: false,
        error: null,
        execute: mockExecute,
      });

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      // Set form data
      act(() => {
        result.current.setNewTransaction(mockNewTransaction);
      });

      // Create transaction
      await act(async () => {
        await result.current.handleCreateTransaction({ preventDefault: jest.fn() });
      });

      expect(api.post).toHaveBeenCalledWith('/transaction', mockNewTransaction);
      expect(mockOnDataChange).toHaveBeenCalled();
      expect(result.current.isTransactionModalOpen).toBe(false);
    });

    test('handleCreateTransaction shows error on API failure', async () => {
      const mockError = {
        response: {
          data: {
            user_message: 'Transaction creation failed',
          },
        },
      };

      api.post.mockRejectedValue(mockError);

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      await act(async () => {
        await result.current.handleCreateTransaction({ preventDefault: jest.fn() });
      });

      expect(global.alert).toHaveBeenCalledWith('Transaction creation failed');
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });

    test('handleCreateTransaction falls back to generic error message', async () => {
      const mockError = new Error('Network error');
      api.post.mockRejectedValue(mockError);

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      await act(async () => {
        await result.current.handleCreateTransaction({ preventDefault: jest.fn() });
      });

      expect(global.alert).toHaveBeenCalledWith('Error creating transaction');
    });
  });

  describe('Update Transaction', () => {
    test('handleUpdateTransaction updates transaction and closes modal', async () => {
      const mockTransaction = {
        id: 1,
        portfolioFundId: 'fund-123',
        date: '2024-01-15',
        type: 'buy',
        shares: 150,
        costPerShare: 55,
      };

      const mockUpdatedTransaction = { ...mockTransaction };

      api.put.mockResolvedValue({ data: mockUpdatedTransaction });

      const mockExistingTransactions = [
        { id: 1, shares: 100 },
        { id: 2, shares: 200 },
      ];

      useApiState.mockReturnValue({
        data: mockExistingTransactions,
        loading: false,
        error: null,
        execute: mockExecute,
      });

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      // Set editing transaction
      act(() => {
        result.current.setEditingTransaction(mockTransaction);
      });

      // Update transaction
      await act(async () => {
        await result.current.handleUpdateTransaction({ preventDefault: jest.fn() });
      });

      expect(api.put).toHaveBeenCalledWith(`/transaction/${mockTransaction.id}`, mockTransaction);
      expect(mockOnDataChange).toHaveBeenCalled();
      expect(result.current.isTransactionEditModalOpen).toBe(false);
      expect(result.current.editingTransaction).toBeNull();
    });

    test('handleUpdateTransaction shows error on API failure', async () => {
      const mockError = {
        response: {
          data: {
            user_message: 'Update failed',
          },
        },
      };

      api.put.mockRejectedValue(mockError);

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setEditingTransaction({ id: 1, shares: 100 });
      });

      await act(async () => {
        await result.current.handleUpdateTransaction({ preventDefault: jest.fn() });
      });

      expect(global.alert).toHaveBeenCalledWith('Update failed');
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });
  });

  describe('Delete Transaction', () => {
    test('handleDeleteTransaction deletes transaction after confirmation', async () => {
      api.delete.mockResolvedValue({});

      const mockTransactions = [
        { id: 1, shares: 100 },
        { id: 2, shares: 200 },
      ];

      useApiState.mockReturnValue({
        data: mockTransactions,
        loading: false,
        error: null,
        execute: mockExecute,
      });

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      await act(async () => {
        await result.current.handleDeleteTransaction(1);
      });

      expect(global.window.confirm).toHaveBeenCalledWith(
        'Are you sure you want to delete this transaction?'
      );
      expect(api.delete).toHaveBeenCalledWith('/transaction/1');
      expect(mockOnDataChange).toHaveBeenCalled();
    });

    test('handleDeleteTransaction does not delete if user cancels', async () => {
      global.window.confirm = jest.fn(() => false);

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      await act(async () => {
        await result.current.handleDeleteTransaction(1);
      });

      expect(api.delete).not.toHaveBeenCalled();
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });

    test('handleDeleteTransaction shows error on API failure', async () => {
      const mockError = {
        response: {
          data: {
            user_message: 'Delete failed',
          },
        },
      };

      api.delete.mockRejectedValue(mockError);

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      await act(async () => {
        await result.current.handleDeleteTransaction(1);
      });

      expect(global.alert).toHaveBeenCalledWith('Delete failed');
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });
  });

  describe('Price Lookup', () => {
    test('handleTransactionDateChange sets date for sell transactions without price lookup', async () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setNewTransaction({
          portfolioFundId: 'fund-123',
          type: 'sell',
          date: getTodayString(),
          shares: '',
          costPerShare: '',
        });
      });

      await act(async () => {
        await result.current.handleTransactionDateChange({ target: { value: '2024-01-15' } }, []);
      });

      expect(result.current.newTransaction.date).toBe('2024-01-15');
      expect(api.get).not.toHaveBeenCalled();
      expect(result.current.priceFound).toBe(false);
    });

    test('handleTransactionDateChange fetches price for buy transactions', async () => {
      const mockPrices = [
        { date: '2024-01-15T00:00:00Z', price: 50.25 },
        { date: '2024-01-16T00:00:00Z', price: 51.0 },
      ];

      api.get.mockResolvedValue({ data: mockPrices });

      const mockPortfolioFunds = [{ id: 'pf-1', fundId: 'fund-123' }];

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setNewTransaction({
          portfolioFundId: 'pf-1',
          type: 'buy',
          date: getTodayString(),
          shares: '',
          costPerShare: '',
        });
      });

      await act(async () => {
        await result.current.handleTransactionDateChange(
          { target: { value: '2024-01-15' } },
          mockPortfolioFunds
        );
      });

      expect(api.get).toHaveBeenCalledWith('/fund/fund-prices/fund-123');
      expect(result.current.newTransaction.date).toBe('2024-01-15');
      expect(result.current.newTransaction.costPerShare).toBe(50.25);
      expect(result.current.priceFound).toBe(true);
    });

    test('handleTransactionDateChange uses cached prices on subsequent calls', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const mockPortfolioFunds = [{ id: 'pf-1', fundId: 'fund-123' }];

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setNewTransaction({
          portfolioFundId: 'pf-1',
          type: 'buy',
          date: getTodayString(),
          shares: '',
          costPerShare: '',
        });
      });

      // First call - should fetch
      await act(async () => {
        await result.current.handleTransactionDateChange(
          { target: { value: '2024-01-15' } },
          mockPortfolioFunds
        );
      });

      expect(api.get).toHaveBeenCalledTimes(1);

      // Second call - should use cache
      await act(async () => {
        await result.current.handleTransactionDateChange(
          { target: { value: '2024-01-15' } },
          mockPortfolioFunds
        );
      });

      expect(api.get).toHaveBeenCalledTimes(1); // Still only called once
      expect(result.current.priceFound).toBe(true);
    });

    test('handleTransactionDateChange handles missing price for date', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const mockPortfolioFunds = [{ id: 'pf-1', fundId: 'fund-123' }];

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setNewTransaction({
          portfolioFundId: 'pf-1',
          type: 'buy',
          date: getTodayString(),
          shares: '',
          costPerShare: '',
        });
      });

      await act(async () => {
        await result.current.handleTransactionDateChange(
          { target: { value: '2024-01-16' } }, // Date not in prices
          mockPortfolioFunds
        );
      });

      expect(result.current.newTransaction.date).toBe('2024-01-16');
      expect(result.current.newTransaction.costPerShare).toBe('');
      expect(result.current.priceFound).toBe(false);
    });

    test('handleTransactionDateChange handles price fetch error', async () => {
      api.get.mockRejectedValue(new Error('Price fetch failed'));

      const mockPortfolioFunds = [{ id: 'pf-1', fundId: 'fund-123' }];

      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setNewTransaction({
          portfolioFundId: 'pf-1',
          type: 'buy',
          date: getTodayString(),
          shares: '',
          costPerShare: '',
        });
      });

      await act(async () => {
        await result.current.handleTransactionDateChange(
          { target: { value: '2024-01-15' } },
          mockPortfolioFunds
        );
      });

      expect(result.current.newTransaction.date).toBe('2024-01-15');
      expect(result.current.priceFound).toBe(false);
      expect(global.console.error).toHaveBeenCalled();
    });
  });

  describe('State Setters', () => {
    test('setNewTransaction updates new transaction state', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      const newData = {
        portfolioFundId: 'fund-123',
        date: '2024-01-15',
        type: 'buy',
        shares: 100,
        costPerShare: 50,
      };

      act(() => {
        result.current.setNewTransaction(newData);
      });

      expect(result.current.newTransaction).toEqual(newData);
    });

    test('setEditingTransaction updates editing transaction state', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      const editData = {
        id: 1,
        portfolioFundId: 'fund-123',
        date: '2024-01-15',
        type: 'buy',
        shares: 100,
        costPerShare: 50,
      };

      act(() => {
        result.current.setEditingTransaction(editData);
      });

      expect(result.current.editingTransaction).toEqual(editData);
    });

    test('setPriceFound updates price found state', () => {
      const { result } = renderHook(() =>
        useTransactionManagement(mockPortfolioId, mockOnDataChange)
      );

      act(() => {
        result.current.setPriceFound(true);
      });

      expect(result.current.priceFound).toBe(true);

      act(() => {
        result.current.setPriceFound(false);
      });

      expect(result.current.priceFound).toBe(false);
    });
  });
});
