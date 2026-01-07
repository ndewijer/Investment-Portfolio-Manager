import { useState, useEffect, useCallback, useRef } from 'react';
import api from '../utils/api';

/**
 * Custom hook for managing chart data with intelligent progressive loading
 *
 * Implements a progressive data loading strategy for time-series charts. Instead of
 * loading all historical data upfront, it loads a default time range initially and
 * fetches additional data as the user zooms out. This significantly improves initial
 * load times while maintaining smooth zoom interactions. The hook tracks loaded ranges
 * and intelligently appends data at boundaries to avoid duplicate API calls.
 *
 * @param {string} endpoint - API endpoint to fetch time-series data from
 * @param {Object} [params={}] - Additional query parameters for the API call
 * @param {number} [defaultZoomDays=365] - Initial number of days to load (default: 1 year)
 * @returns {Object} Chart data management object
 * @returns {Array} returns.data - Array of data points sorted by date
 * @returns {boolean} returns.loading - True when fetching data from the API
 * @returns {string|null} returns.error - Error message if data fetch failed
 * @returns {Object|null} returns.loadedRange - Currently loaded date range {start, end}
 * @returns {Object|null} returns.totalDataRange - Total available data range
 * @returns {function} returns.onZoomChange - Callback for chart zoom state changes
 * @returns {function} returns.loadDateRange - Manually load a specific date range
 * @returns {function} returns.resetToInitialRange - Reset to the default zoom level
 * @returns {function} returns.loadAllData - Load all available historical data
 * @returns {function} returns.refetch - Re-fetch the currently loaded range
 *
 * @example
 * const {
 *   data,
 *   loading,
 *   onZoomChange,
 *   loadAllData
 * } = useChartData('/portfolio/history', { portfolio_id: 123 }, 180);
 *
 * return (
 *   <Chart
 *     data={data}
 *     loading={loading}
 *     onZoom={onZoomChange}
 *     onLoadAll={loadAllData}
 *   />
 * );
 *
 * @see PortfolioChart for implementation example
 */
const useChartData = (endpoint, params = {}, defaultZoomDays = 365) => {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [loadedRange, setLoadedRange] = useState(null);
  const [totalDataRange, setTotalDataRange] = useState(null);

  // Keep track of the current zoom state to determine when to load more data
  const currentZoomRef = useRef(null);
  const isLoadingRef = useRef(false);
  const initializedRef = useRef(false);

  // Fetch data for a specific date range
  const fetchDataRange = useCallback(
    async (startDate, endDate, append = false) => {
      if (isLoadingRef.current) return;

      try {
        isLoadingRef.current = true;
        setLoading(true);
        setError(null);

        const queryParams = new URLSearchParams({
          ...params,
          ...(startDate && { start_date: startDate }),
          ...(endDate && { end_date: endDate }),
        });

        const response = await api.get(`${endpoint}?${queryParams}`);
        const newData = response.data;

        if (append) {
          // Merge new data with existing data, avoiding duplicates
          setData((prevData) => {
            const existingDates = new Set(prevData.map((item) => item.date));
            const uniqueNewData = newData.filter((item) => !existingDates.has(item.date));

            // Sort combined data by date
            const combined = [...prevData, ...uniqueNewData];
            return combined.sort((a, b) => new Date(a.date) - new Date(b.date));
          });
        } else {
          setData(newData);
        }

        // Update loaded range
        const newLoadedRange = {
          start: startDate || (newData.length > 0 ? newData[0].date : null),
          end: endDate || (newData.length > 0 ? newData[newData.length - 1].date : null),
        };

        if (append) {
          setLoadedRange((prevRange) => {
            if (!prevRange) return newLoadedRange;
            return {
              start:
                startDate && new Date(startDate) < new Date(prevRange.start)
                  ? startDate
                  : prevRange.start,
              end: endDate && new Date(endDate) > new Date(prevRange.end) ? endDate : prevRange.end,
            };
          });
        } else {
          setLoadedRange(newLoadedRange);
        }

        // Set total data range on first load if not already set
        setTotalDataRange((prevRange) => {
          if (prevRange || newData.length === 0) return prevRange;

          // For now, we'll estimate the total range based on the current data
          return {
            start: newData[0].date,
            end: newData[newData.length - 1].date,
          };
        });
      } catch (err) {
        setError('Error fetching chart data');
        console.error('Error fetching chart data:', err);
      } finally {
        setLoading(false);
        isLoadingRef.current = false;
      }
    },
    [endpoint, params]
  );

  // Initial data load - only run once when the hook is first used
  useEffect(() => {
    if (!initializedRef.current) {
      initializedRef.current = true;

      const endDate = new Date();
      const startDate = new Date();
      startDate.setDate(endDate.getDate() - defaultZoomDays);

      fetchDataRange(startDate.toISOString().split('T')[0], endDate.toISOString().split('T')[0]);
    }
  }, [defaultZoomDays, fetchDataRange]);

  // Check if we need to load more data based on zoom state
  const checkAndLoadMoreData = useCallback(
    (zoomState) => {
      if (!loadedRange || !data.length || isLoadingRef.current) return;

      currentZoomRef.current = zoomState;

      // If zooming out and approaching the boundaries of loaded data, load more
      if (zoomState.isZoomed && zoomState.xDomain) {
        const [startIndex, endIndex] = zoomState.xDomain;
        const dataLength = data.length;

        // Calculate buffer zone (10% of current data range)
        const bufferSize = Math.max(10, Math.floor(dataLength * 0.1));

        // Check if we're approaching the start boundary
        if (startIndex < bufferSize) {
          const currentStartDate = new Date(loadedRange.start);
          const extendDays = Math.max(30, defaultZoomDays / 2); // Load at least 30 days or half the default zoom
          const newStartDate = new Date(currentStartDate);
          newStartDate.setDate(currentStartDate.getDate() - extendDays);

          fetchDataRange(
            newStartDate.toISOString().split('T')[0],
            loadedRange.start,
            true // append to existing data
          );
        }

        // Check if we're approaching the end boundary
        if (endIndex > dataLength - bufferSize) {
          const currentEndDate = new Date(loadedRange.end);
          const extendDays = Math.max(30, defaultZoomDays / 2);
          const newEndDate = new Date(currentEndDate);
          newEndDate.setDate(currentEndDate.getDate() + extendDays);
          const today = new Date();

          // Don't extend beyond today
          if (newEndDate <= today) {
            fetchDataRange(
              loadedRange.end,
              newEndDate.toISOString().split('T')[0],
              true // append to existing data
            );
          }
        }
      }
    },
    [data, loadedRange, defaultZoomDays, fetchDataRange]
  );

  // Function to handle zoom state changes from the chart
  const onZoomChange = useCallback(
    (zoomState) => {
      // Debounce the check to avoid too many API calls
      setTimeout(() => {
        checkAndLoadMoreData(zoomState);
      }, 100);
    },
    [checkAndLoadMoreData]
  );

  // Function to manually load a specific date range
  const loadDateRange = useCallback(
    (startDate, endDate) => {
      fetchDataRange(startDate, endDate, false);
    },
    [fetchDataRange]
  );

  // Function to reset to initial data range
  const resetToInitialRange = useCallback(() => {
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(endDate.getDate() - defaultZoomDays);

    fetchDataRange(
      startDate.toISOString().split('T')[0],
      endDate.toISOString().split('T')[0],
      false
    );
  }, [defaultZoomDays, fetchDataRange]);

  // Function to load all available data
  const loadAllData = useCallback(() => {
    // Only fetch if we don't already have all data
    if (!totalDataRange || !totalDataRange.isFullDataset) {
      // Load all data by not specifying date range
      fetchDataRange(null, null, false);
      // Mark that we've loaded all data to prevent initial zoom from being reapplied
      setTotalDataRange({ isFullDataset: true });
    }
  }, [fetchDataRange, totalDataRange]);

  return {
    data,
    loading,
    error,
    loadedRange,
    totalDataRange,
    onZoomChange,
    loadDateRange,
    resetToInitialRange,
    loadAllData,
    refetch: () => {
      if (loadedRange) {
        fetchDataRange(loadedRange.start, loadedRange.end, false);
      } else {
        resetToInitialRange();
      }
    },
  };
};

export default useChartData;
