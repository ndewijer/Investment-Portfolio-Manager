import { useState, useCallback, useRef } from 'react';

/**
 * Custom hook for managing API state consistently across components
 *
 * Provides a standardized pattern for handling API calls with loading, error, and data states.
 * This hook simplifies async operations by abstracting common patterns like error handling,
 * loading states, and data updates. Use this when you need to fetch data from an API endpoint
 * and manage the associated UI states.
 *
 * @param {*} [initialData=null] - Initial data value before any API call is made
 * @returns {Object} API state management object
 * @returns {*} returns.data - Current data from the API or initial data
 * @returns {boolean} returns.loading - True when an API call is in progress
 * @returns {string|null} returns.error - Error message if the API call failed, null otherwise
 * @returns {function} returns.execute - Function to execute an API call with options
 * @returns {function} returns.reset - Resets state back to initial values
 * @returns {function} returns.setData - Directly update data without making an API call
 * @returns {function} returns.clearError - Clear the current error state
 *
 * @example
 * const { data, loading, error, execute } = useApiState([]);
 *
 * useEffect(() => {
 *   execute(() => api.get('/portfolios'), {
 *     onSuccess: (data) => console.log('Loaded:', data),
 *     onError: (err) => console.error('Failed:', err)
 *   });
 * }, []);
 *
 * @see FormatContext for formatting utilities often used with API data
 */
const useApiState = (initialData = null) => {
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const initialDataRef = useRef(initialData);

  // Update ref when initialData changes
  initialDataRef.current = initialData;

  const execute = useCallback(async (apiCall, options = {}) => {
    const {
      onSuccess,
      onError,
      showSuccessMessage = false,
      successMessage = 'Operation completed successfully',
      resetOnStart = false,
    } = options;

    try {
      setLoading(true);
      setError(null);

      if (resetOnStart) {
        setData(initialDataRef.current);
      }

      const response = await apiCall();
      setData(response.data || response);

      if (onSuccess) {
        onSuccess(response.data || response);
      }

      if (showSuccessMessage) {
        // This could be enhanced to use a toast notification system
        console.log(successMessage);
      }

      return response.data || response;
    } catch (err) {
      const errorMessage =
        err.response?.data?.user_message ||
        err.response?.data?.message ||
        err.response?.data?.error ||
        err.message ||
        'An unexpected error occurred';

      setError(errorMessage);

      if (onError) {
        onError(err, errorMessage);
      } else {
        console.error('API Error:', err);
      }

      throw err;
    } finally {
      setLoading(false);
    }
  }, []); // Include initialData dependency as required by ESLint

  const reset = useCallback(() => {
    setData(initialDataRef.current);
    setLoading(false);
    setError(null);
  }, []);

  const setDataDirectly = useCallback((newData) => {
    setData(newData);
    setError(null);
  }, []);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    data,
    loading,
    error,
    execute,
    reset,
    setData: setDataDirectly,
    clearError,
  };
};

export default useApiState;
