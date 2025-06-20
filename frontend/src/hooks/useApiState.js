import { useState, useCallback } from 'react';

/**
 * Custom hook for managing API state consistently across components
 * Provides loading, error, and data states with standardized error handling
 */
const useApiState = (initialData = null) => {
  const [data, setData] = useState(initialData);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const execute = useCallback(
    async (apiCall, options = {}) => {
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
          setData(initialData);
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
    },
    [initialData]
  ); // Remove initialData dependency to prevent recreation

  const reset = useCallback(() => {
    setData(initialData);
    setLoading(false);
    setError(null);
  }, [initialData]);

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
