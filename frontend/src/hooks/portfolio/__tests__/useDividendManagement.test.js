/**
 * @fileoverview Test suite for useDividendManagement custom hook
 *
 * Tests dividend CRUD operations and state management:
 * - Modal state management (open/close for create and edit)
 * - Create dividend with stock/cash dividend logic
 * - Edit dividend workflow with transaction lookup
 * - Update dividend with validation
 * - Delete dividend with confirmation
 * - Stock dividend validation (buy order date, reinvestment fields)
 * - Future vs past dividend handling
 *
 * Testing approach:
 * - Mock useApiState hook for dividend data management
 * - Mock API calls (api.post, api.put, api.delete, api.get)
 * - Mock window.confirm and window.alert
 * - Mock date helper functions
 * - Use renderHook and act from @testing-library/react
 * - Target coverage: 70%+ focusing on critical business logic
 *
 * Total: 30+ tests
 */
import { renderHook, act } from '@testing-library/react';
import { useDividendManagement } from '../useDividendManagement';
import useApiState from '../../useApiState';
import api from '../../../utils/api';
import * as dateHelpers from '../../../utils/portfolio/dateHelpers';

// Mock dependencies
jest.mock('../../useApiState');
jest.mock('../../../utils/api');
jest.mock('../../../utils/portfolio/dateHelpers');

describe('useDividendManagement', () => {
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

    // Mock window methods
    global.window.confirm = jest.fn(() => true);
    global.window.alert = jest.fn();

    // Mock console methods
    global.console.error = jest.fn();

    // Mock date helpers
    dateHelpers.getTodayString.mockReturnValue('2024-01-15');
    dateHelpers.toDateString.mockImplementation((date) => {
      if (typeof date === 'string') return date.split('T')[0];
      return date;
    });
    dateHelpers.isDateInFuture.mockReturnValue(false);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Initialization', () => {
    test('initializes with correct default state', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      expect(result.current.dividends).toEqual([]);
      expect(result.current.dividendsLoading).toBe(false);
      expect(result.current.dividendsError).toBeNull();
      expect(result.current.isDividendModalOpen).toBe(false);
      expect(result.current.isDividendEditModalOpen).toBe(false);
      expect(result.current.editingDividend).toBeNull();
      expect(result.current.selectedFund).toBeNull();
    });

    test('initializes newDividend with correct defaults', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      expect(result.current.newDividend).toMatchObject({
        portfolio_fund_id: '',
        record_date: '2024-01-15',
        ex_dividend_date: '2024-01-15',
        dividend_per_share: '',
        buy_order_date: '',
        reinvestment_shares: '',
        reinvestment_price: '',
      });
    });
  });

  describe('Load Dividends', () => {
    test('loadDividends fetches dividends for portfolio', async () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.loadDividends();
      });

      expect(mockExecute).toHaveBeenCalled();
    });

    test('loadDividends does nothing if no portfolioId', async () => {
      const { result } = renderHook(() => useDividendManagement(null, mockOnDataChange));

      await act(async () => {
        await result.current.loadDividends();
      });

      expect(mockExecute).not.toHaveBeenCalled();
    });
  });

  describe('Add Dividend Modal', () => {
    test('handleAddDividend fetches fund details and opens modal', async () => {
      const mockFund = { id: 'pf-1', fund_id: 'fund-123' };
      const mockFundData = { id: 'fund-123', name: 'Test Fund', dividend_type: 'cash' };

      api.get.mockResolvedValue({ data: mockFundData });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleAddDividend(mockFund);
      });

      expect(api.get).toHaveBeenCalledWith('/funds/fund-123');
      expect(result.current.isDividendModalOpen).toBe(true);
      expect(result.current.selectedFund).toEqual(mockFundData);
      expect(result.current.newDividend.portfolio_fund_id).toBe('pf-1');
    });

    test('handleAddDividend handles API error gracefully', async () => {
      const mockFund = { id: 'pf-1', fund_id: 'fund-123' };
      api.get.mockRejectedValue(new Error('Fund not found'));

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleAddDividend(mockFund);
      });

      expect(global.console.error).toHaveBeenCalled();
      expect(result.current.isDividendModalOpen).toBe(false);
    });

    test('closeDividendModal resets form and closes modal', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      // Set some state first
      act(() => {
        result.current.setSelectedFund({ id: 'fund-123' });
      });

      act(() => {
        result.current.closeDividendModal();
      });

      expect(result.current.isDividendModalOpen).toBe(false);
      expect(result.current.selectedFund).toBeNull();
      expect(result.current.newDividend.portfolio_fund_id).toBe('');
    });
  });

  describe('Create Dividend - Cash', () => {
    test('handleCreateDividend creates cash dividend successfully', async () => {
      const mockDividendData = {
        portfolio_fund_id: 'pf-1',
        record_date: '2024-01-15',
        ex_dividend_date: '2024-01-15',
        dividend_per_share: '1.50',
      };

      const mockCreatedDividend = { ...mockDividendData, id: 1 };

      api.post.mockResolvedValue({ data: mockCreatedDividend });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      // Set up as cash dividend
      act(() => {
        result.current.setSelectedFund({ dividend_type: 'cash' });
        result.current.setNewDividend(mockDividendData);
      });

      await act(async () => {
        await result.current.handleCreateDividend({ preventDefault: jest.fn() });
      });

      expect(api.post).toHaveBeenCalledWith(
        '/dividends',
        expect.objectContaining({
          portfolio_fund_id: 'pf-1',
          dividend_per_share: '1.50',
        })
      );
      expect(mockOnDataChange).toHaveBeenCalled();
      expect(result.current.isDividendModalOpen).toBe(false);
    });

    test('handleCreateDividend shows error on API failure', async () => {
      const mockError = {
        response: {
          data: {
            user_message: 'Dividend creation failed',
          },
        },
      };

      api.post.mockRejectedValue(mockError);

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setSelectedFund({ dividend_type: 'cash' });
      });

      await act(async () => {
        await result.current.handleCreateDividend({ preventDefault: jest.fn() });
      });

      expect(global.window.alert).toHaveBeenCalledWith('Dividend creation failed');
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });
  });

  describe('Create Dividend - Stock (Future)', () => {
    test('creates stock dividend with future buy order date', async () => {
      dateHelpers.isDateInFuture.mockReturnValue(true);

      const mockDividendData = {
        portfolio_fund_id: 'pf-1',
        record_date: '2024-01-15',
        ex_dividend_date: '2024-01-15',
        dividend_per_share: '1.50',
        buy_order_date: '2024-02-01',
      };

      api.post.mockResolvedValue({ data: { ...mockDividendData, id: 1 } });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setSelectedFund({ dividend_type: 'stock' });
        result.current.setNewDividend(mockDividendData);
      });

      await act(async () => {
        await result.current.handleCreateDividend({ preventDefault: jest.fn() });
      });

      expect(api.post).toHaveBeenCalledWith(
        '/dividends',
        expect.objectContaining({
          buy_order_date: '2024-02-01',
          reinvestment_shares: undefined,
          reinvestment_price: undefined,
        })
      );
      expect(mockOnDataChange).toHaveBeenCalled();
    });

    test('validates buy_order_date is required for stock dividends', async () => {
      // Mock as future date to bypass reinvestment validation
      dateHelpers.isDateInFuture.mockReturnValue(true);

      const mockDividendData = {
        portfolio_fund_id: 'pf-1',
        dividend_per_share: '1.50',
        buy_order_date: '', // Missing
      };

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setSelectedFund({ dividend_type: 'stock' });
        result.current.setNewDividend(mockDividendData);
      });

      await act(async () => {
        await result.current.handleCreateDividend({ preventDefault: jest.fn() });
      });

      expect(global.window.alert).toHaveBeenCalledWith(
        'Please specify a buy order date for stock dividends'
      );
      expect(api.post).not.toHaveBeenCalled();
    });
  });

  describe('Create Dividend - Stock (Past)', () => {
    test('creates stock dividend with past buy order date and reinvestment', async () => {
      dateHelpers.isDateInFuture.mockReturnValue(false);

      const mockDividendData = {
        portfolio_fund_id: 'pf-1',
        record_date: '2024-01-15',
        ex_dividend_date: '2024-01-15',
        dividend_per_share: '1.50',
        buy_order_date: '2024-01-10',
        reinvestment_shares: '10',
        reinvestment_price: '50.00',
      };

      api.post.mockResolvedValue({ data: { ...mockDividendData, id: 1 } });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setSelectedFund({ dividend_type: 'stock' });
        result.current.setNewDividend(mockDividendData);
      });

      await act(async () => {
        await result.current.handleCreateDividend({ preventDefault: jest.fn() });
      });

      expect(api.post).toHaveBeenCalledWith(
        '/dividends',
        expect.objectContaining({
          buy_order_date: '2024-01-10',
          reinvestment_shares: '10',
          reinvestment_price: '50.00',
        })
      );
      expect(mockOnDataChange).toHaveBeenCalled();
    });

    test('validates reinvestment fields for past stock dividends', async () => {
      dateHelpers.isDateInFuture.mockReturnValue(false);

      const mockDividendData = {
        portfolio_fund_id: 'pf-1',
        dividend_per_share: '1.50',
        buy_order_date: '2024-01-10',
        reinvestment_shares: '', // Missing
        reinvestment_price: '', // Missing
      };

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setSelectedFund({ dividend_type: 'stock' });
        result.current.setNewDividend(mockDividendData);
      });

      await act(async () => {
        await result.current.handleCreateDividend({ preventDefault: jest.fn() });
      });

      expect(global.window.alert).toHaveBeenCalledWith(
        'Please fill in both reinvestment shares and price for completed stock dividends'
      );
      expect(api.post).not.toHaveBeenCalled();
    });
  });

  describe('Edit Dividend', () => {
    test('handleEditDividend fetches fund and opens edit modal', async () => {
      const mockDividend = {
        id: 1,
        fund_id: 'fund-123',
        record_date: '2024-01-15T00:00:00Z',
        ex_dividend_date: '2024-01-15T00:00:00Z',
        dividend_per_share: '1.50',
      };

      const mockFundData = { id: 'fund-123', name: 'Test Fund', dividend_type: 'cash' };

      api.get.mockResolvedValue({ data: mockFundData });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleEditDividend(mockDividend);
      });

      expect(api.get).toHaveBeenCalledWith('/funds/fund-123');
      expect(result.current.isDividendEditModalOpen).toBe(true);
      expect(result.current.selectedFund).toEqual(mockFundData);
      expect(result.current.editingDividend).toMatchObject({
        id: 1,
        record_date: '2024-01-15',
        ex_dividend_date: '2024-01-15',
      });
    });

    test('handleEditDividend fetches transaction for stock dividend with reinvestment', async () => {
      const mockDividend = {
        id: 1,
        fund_id: 'fund-123',
        record_date: '2024-01-15T00:00:00Z',
        ex_dividend_date: '2024-01-15T00:00:00Z',
        reinvestment_transaction_id: 'txn-456',
      };

      const mockFundData = { id: 'fund-123', dividend_type: 'stock' };
      const mockTransaction = { id: 'txn-456', shares: 10, cost_per_share: 50 };

      api.get
        .mockResolvedValueOnce({ data: mockFundData })
        .mockResolvedValueOnce({ data: mockTransaction });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleEditDividend(mockDividend);
      });

      expect(api.get).toHaveBeenCalledWith('/funds/fund-123');
      expect(api.get).toHaveBeenCalledWith('/transactions/txn-456');
      expect(result.current.editingDividend.reinvestment_shares).toBe(10);
      expect(result.current.editingDividend.reinvestment_price).toBe(50);
    });

    test('handleEditDividend handles API error', async () => {
      api.get.mockRejectedValue(new Error('Fund not found'));

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleEditDividend({ id: 1, fund_id: 'fund-123' });
      });

      expect(global.window.alert).toHaveBeenCalledWith('Error loading dividend details');
      expect(result.current.isDividendEditModalOpen).toBe(false);
    });

    test('closeDividendEditModal resets state', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setEditingDividend({ id: 1 });
        result.current.setSelectedFund({ id: 'fund-123' });
      });

      act(() => {
        result.current.closeDividendEditModal();
      });

      expect(result.current.isDividendEditModalOpen).toBe(false);
      expect(result.current.editingDividend).toBeNull();
      expect(result.current.selectedFund).toBeNull();
    });
  });

  describe('Update Dividend', () => {
    test('handleUpdateDividend updates cash dividend', async () => {
      const mockDividend = {
        id: 1,
        portfolio_fund_id: 'pf-1',
        dividend_per_share: '2.00',
      };

      api.put.mockResolvedValue({ data: mockDividend });

      const mockExistingDividends = [{ id: 1, dividend_per_share: '1.50' }];

      useApiState.mockReturnValue({
        data: mockExistingDividends,
        loading: false,
        error: null,
        execute: mockExecute,
      });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setEditingDividend(mockDividend);
        result.current.setSelectedFund({ dividend_type: 'cash' });
      });

      await act(async () => {
        await result.current.handleUpdateDividend({ preventDefault: jest.fn() });
      });

      expect(api.put).toHaveBeenCalledWith(
        '/dividends/1',
        expect.objectContaining({ id: 1, dividend_per_share: '2.00' })
      );
      expect(mockOnDataChange).toHaveBeenCalled();
      expect(result.current.isDividendEditModalOpen).toBe(false);
    });

    test('handleUpdateDividend validates stock dividend reinvestment fields', async () => {
      const mockDividend = {
        id: 1,
        reinvestment_shares: '', // Missing
        reinvestment_price: '', // Missing
      };

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setEditingDividend(mockDividend);
        result.current.setSelectedFund({ dividend_type: 'stock' });
      });

      await act(async () => {
        await result.current.handleUpdateDividend({ preventDefault: jest.fn() });
      });

      expect(global.window.alert).toHaveBeenCalledWith(
        'Please fill in both reinvestment shares and price for stock dividends'
      );
      expect(api.put).not.toHaveBeenCalled();
    });

    test('handleUpdateDividend handles API error', async () => {
      const mockError = {
        response: {
          data: {
            error: 'Update failed',
          },
        },
      };

      api.put.mockRejectedValue(mockError);

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      act(() => {
        result.current.setEditingDividend({ id: 1 });
        result.current.setSelectedFund({ dividend_type: 'cash' });
      });

      await act(async () => {
        await result.current.handleUpdateDividend({ preventDefault: jest.fn() });
      });

      expect(global.window.alert).toHaveBeenCalledWith('Update failed');
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });
  });

  describe('Delete Dividend', () => {
    test('handleDeleteDividend deletes dividend after confirmation', async () => {
      api.delete.mockResolvedValue({});

      const mockDividends = [
        { id: 1, dividend_per_share: '1.50' },
        { id: 2, dividend_per_share: '2.00' },
      ];

      useApiState.mockReturnValue({
        data: mockDividends,
        loading: false,
        error: null,
        execute: mockExecute,
      });

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleDeleteDividend(1);
      });

      expect(global.window.confirm).toHaveBeenCalledWith(
        'Are you sure you want to delete this dividend?'
      );
      expect(api.delete).toHaveBeenCalledWith('/dividends/1');
      expect(mockOnDataChange).toHaveBeenCalled();
    });

    test('handleDeleteDividend does not delete if user cancels', async () => {
      global.window.confirm = jest.fn(() => false);

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleDeleteDividend(1);
      });

      expect(api.delete).not.toHaveBeenCalled();
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });

    test('handleDeleteDividend handles API error', async () => {
      const mockError = {
        response: {
          data: {
            user_message: 'Delete failed',
          },
        },
      };

      api.delete.mockRejectedValue(mockError);

      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      await act(async () => {
        await result.current.handleDeleteDividend(1);
      });

      expect(global.window.alert).toHaveBeenCalledWith('Delete failed');
      expect(mockOnDataChange).not.toHaveBeenCalled();
    });
  });

  describe('State Setters', () => {
    test('setNewDividend updates new dividend state', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      const newData = {
        portfolio_fund_id: 'pf-1',
        dividend_per_share: '1.50',
      };

      act(() => {
        result.current.setNewDividend(newData);
      });

      expect(result.current.newDividend).toEqual(newData);
    });

    test('setEditingDividend updates editing dividend state', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      const editData = {
        id: 1,
        dividend_per_share: '2.00',
      };

      act(() => {
        result.current.setEditingDividend(editData);
      });

      expect(result.current.editingDividend).toEqual(editData);
    });

    test('setSelectedFund updates selected fund state', () => {
      const { result } = renderHook(() => useDividendManagement(mockPortfolioId, mockOnDataChange));

      const fundData = {
        id: 'fund-123',
        name: 'Test Fund',
        dividend_type: 'stock',
      };

      act(() => {
        result.current.setSelectedFund(fundData);
      });

      expect(result.current.selectedFund).toEqual(fundData);
    });
  });
});
