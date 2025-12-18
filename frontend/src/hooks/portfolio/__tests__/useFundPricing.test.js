/**
 * @fileoverview Test suite for useFundPricing custom hook
 *
 * Tests fund pricing data management with caching:
 * - Price fetching and caching
 * - Date-based price lookup
 * - Cache hit/miss behavior
 * - Price found indicator management
 * - Cache clearing operations
 *
 * Testing approach:
 * - Mock API calls (api.get)
 * - Mock console.error for error cases
 * - Use renderHook and act from @testing-library/react
 * - Test caching behavior to verify efficiency
 * - Target coverage: 70%+ focusing on cache logic
 *
 * Total: 20+ tests
 */
import { renderHook, act } from '@testing-library/react';
import { useFundPricing } from '../useFundPricing';
import api from '../../../utils/api';

// Mock dependencies
jest.mock('../../../utils/api');

describe('useFundPricing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    global.console.error = jest.fn();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('Initialization', () => {
    test('initializes with correct default state', () => {
      const { result } = renderHook(() => useFundPricing());

      expect(result.current.fundPrices).toEqual({});
      expect(result.current.priceFound).toBe(false);
    });

    test('provides all expected functions', () => {
      const { result } = renderHook(() => useFundPricing());

      expect(typeof result.current.fetchFundPrice).toBe('function');
      expect(typeof result.current.getFundPriceForDate).toBe('function');
      expect(typeof result.current.hasPriceForDate).toBe('function');
      expect(typeof result.current.clearPriceFound).toBe('function');
      expect(typeof result.current.resetFundPrices).toBe('function');
      expect(typeof result.current.setFundPrices).toBe('function');
      expect(typeof result.current.setPriceFound).toBe('function');
    });
  });

  describe('fetchFundPrice', () => {
    test('fetches and transforms prices into date-indexed map', async () => {
      const mockPrices = [
        { date: '2024-01-15T00:00:00Z', price: 50.25 },
        { date: '2024-01-16T00:00:00Z', price: 51.0 },
        { date: '2024-01-17T00:00:00Z', price: 49.75 },
      ];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      let priceMap;
      await act(async () => {
        priceMap = await result.current.fetchFundPrice('fund-123');
      });

      expect(api.get).toHaveBeenCalledWith('/funds/fund-prices/fund-123');
      expect(priceMap).toEqual({
        '2024-01-15': 50.25,
        '2024-01-16': 51.0,
        '2024-01-17': 49.75,
      });
    });

    test('handles empty price array', async () => {
      api.get.mockResolvedValue({ data: [] });

      const { result } = renderHook(() => useFundPricing());

      let priceMap;
      await act(async () => {
        priceMap = await result.current.fetchFundPrice('fund-123');
      });

      expect(priceMap).toEqual({});
    });

    test('handles API error and returns null', async () => {
      api.get.mockRejectedValue(new Error('API Error'));

      const { result } = renderHook(() => useFundPricing());

      let priceMap;
      await act(async () => {
        priceMap = await result.current.fetchFundPrice('fund-123');
      });

      expect(priceMap).toBeNull();
      expect(global.console.error).toHaveBeenCalledWith(
        'Error fetching fund prices:',
        expect.any(Error)
      );
    });

    test('handles prices with different date formats', async () => {
      const mockPrices = [
        { date: '2024-01-15T10:30:45.123Z', price: 50.25 },
        { date: '2024-01-16', price: 51.0 },
      ];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      let priceMap;
      await act(async () => {
        priceMap = await result.current.fetchFundPrice('fund-123');
      });

      expect(priceMap['2024-01-15']).toBe(50.25);
      expect(priceMap['2024-01-16']).toBe(51.0);
    });
  });

  describe('getFundPriceForDate', () => {
    test('fetches and returns price for specific date', async () => {
      const mockPrices = [
        { date: '2024-01-15T00:00:00Z', price: 50.25 },
        { date: '2024-01-16T00:00:00Z', price: 51.0 },
      ];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      let price;
      await act(async () => {
        price = await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(price).toBe(50.25);
      expect(result.current.priceFound).toBe(true);
    });

    test('caches prices and reuses for subsequent calls', async () => {
      const mockPrices = [
        { date: '2024-01-15T00:00:00Z', price: 50.25 },
        { date: '2024-01-16T00:00:00Z', price: 51.0 },
      ];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      // First call - should fetch
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(api.get).toHaveBeenCalledTimes(1);

      // Second call - should use cache
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-16');
      });

      expect(api.get).toHaveBeenCalledTimes(1); // Still only 1 call
      expect(result.current.priceFound).toBe(true);
    });

    test('returns null and sets priceFound false for missing date', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      let price;
      await act(async () => {
        price = await result.current.getFundPriceForDate('fund-123', '2024-01-20');
      });

      expect(price).toBeNull();
      expect(result.current.priceFound).toBe(false);
    });

    test('handles fetch error gracefully', async () => {
      api.get.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useFundPricing());

      let price;
      await act(async () => {
        price = await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(price).toBeNull();
      expect(result.current.priceFound).toBe(false);
    });

    test('fetches different funds independently', async () => {
      const mockPrices1 = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];
      const mockPrices2 = [{ date: '2024-01-15T00:00:00Z', price: 100.5 }];

      api.get
        .mockResolvedValueOnce({ data: mockPrices1 })
        .mockResolvedValueOnce({ data: mockPrices2 });

      const { result } = renderHook(() => useFundPricing());

      let price1, price2;
      await act(async () => {
        price1 = await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      await act(async () => {
        price2 = await result.current.getFundPriceForDate('fund-456', '2024-01-15');
      });

      expect(price1).toBe(50.25);
      expect(price2).toBe(100.5);
      expect(api.get).toHaveBeenCalledTimes(2);
      expect(api.get).toHaveBeenCalledWith('/funds/fund-prices/fund-123');
      expect(api.get).toHaveBeenCalledWith('/funds/fund-prices/fund-456');
    });
  });

  describe('hasPriceForDate', () => {
    test('returns falsy value for uncached fund', () => {
      const { result } = renderHook(() => useFundPricing());

      const hasPrice = result.current.hasPriceForDate('fund-123', '2024-01-15');

      expect(hasPrice).toBeFalsy();
    });

    test('returns true when price exists in cache', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      // Populate cache
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      // Check cache
      const hasPrice = result.current.hasPriceForDate('fund-123', '2024-01-15');

      expect(hasPrice).toBe(true);
    });

    test('returns false when date not in cache', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      // Populate cache
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      // Check for different date
      const hasPrice = result.current.hasPriceForDate('fund-123', '2024-01-20');

      expect(hasPrice).toBe(false);
    });

    test('does not trigger API call', () => {
      const { result } = renderHook(() => useFundPricing());

      result.current.hasPriceForDate('fund-123', '2024-01-15');

      expect(api.get).not.toHaveBeenCalled();
    });
  });

  describe('clearPriceFound', () => {
    test('resets priceFound to false', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      // Set priceFound to true
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(result.current.priceFound).toBe(true);

      // Clear it
      act(() => {
        result.current.clearPriceFound();
      });

      expect(result.current.priceFound).toBe(false);
    });
  });

  describe('resetFundPrices', () => {
    test('clears all cached prices', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      // Populate cache
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(result.current.fundPrices).toHaveProperty('fund-123');

      // Reset
      act(() => {
        result.current.resetFundPrices();
      });

      expect(result.current.fundPrices).toEqual({});
      expect(result.current.priceFound).toBe(false);
    });

    test('forces new API call after reset', async () => {
      const mockPrices = [{ date: '2024-01-15T00:00:00Z', price: 50.25 }];

      api.get.mockResolvedValue({ data: mockPrices });

      const { result } = renderHook(() => useFundPricing());

      // First call
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(api.get).toHaveBeenCalledTimes(1);

      // Reset
      act(() => {
        result.current.resetFundPrices();
      });

      // Second call - should fetch again
      await act(async () => {
        await result.current.getFundPriceForDate('fund-123', '2024-01-15');
      });

      expect(api.get).toHaveBeenCalledTimes(2);
    });
  });

  describe('State Setters', () => {
    test('setFundPrices updates cache directly', () => {
      const { result } = renderHook(() => useFundPricing());

      const customPrices = {
        'fund-123': {
          '2024-01-15': 50.25,
          '2024-01-16': 51.0,
        },
      };

      act(() => {
        result.current.setFundPrices(customPrices);
      });

      expect(result.current.fundPrices).toEqual(customPrices);
    });

    test('setPriceFound updates price found indicator', () => {
      const { result } = renderHook(() => useFundPricing());

      act(() => {
        result.current.setPriceFound(true);
      });

      expect(result.current.priceFound).toBe(true);

      act(() => {
        result.current.setPriceFound(false);
      });

      expect(result.current.priceFound).toBe(false);
    });

    test('allows manual cache management', () => {
      const { result } = renderHook(() => useFundPricing());

      // Manually set cache
      act(() => {
        result.current.setFundPrices({
          'fund-123': { '2024-01-15': 50.25 },
        });
      });

      // Check with hasPriceForDate
      const hasPrice = result.current.hasPriceForDate('fund-123', '2024-01-15');

      expect(hasPrice).toBe(true);
      expect(api.get).not.toHaveBeenCalled();
    });
  });
});
