/**
 * @file AppContext.simple.test.js
 * @description Simplified test suite for AppContext focusing on core functionality
 */

import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AppProvider, useApp } from '../AppContext';
import api from '../../utils/api';

// Mock the API module
jest.mock('../../utils/api');

// Mock HealthCheckError component
jest.mock('../../components/HealthCheckError', () => {
  return function MockHealthCheckError({ error, onRetry }) {
    return (
      <div data-testid="health-check-error">
        {error}
        <button onClick={onRetry}>Retry</button>
      </div>
    );
  };
});

/**
 * Simple test component
 *
 * @returns {JSX.Element} Test component
 */
const TestComponent = () => {
  const { versionInfo, features, loading, error } = useApp();

  return (
    <div>
      <div data-testid="loading">{loading ? 'true' : 'false'}</div>
      <div data-testid="error">{error || 'none'}</div>
      <div data-testid="app-version">{versionInfo.app_version}</div>
      <div data-testid="db-version">{versionInfo.db_version}</div>
      <div data-testid="ibkr-feature">{features.ibkr_integration ? 'true' : 'false'}</div>
    </div>
  );
};

describe('AppContext - Core Functionality', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    jest.spyOn(console, 'debug').mockImplementation(() => {});
  });

  afterEach(() => {
    console.error.mockRestore();
    console.debug.mockRestore();
  });

  describe('Provider and Hook', () => {
    test('provides context to children when healthy', async () => {
      api.get
        .mockResolvedValueOnce({ data: { status: 'healthy' } }) // Health check
        .mockResolvedValueOnce({
          // Version info
          data: {
            app_version: '1.3.5',
            db_version: '1.3.3',
            features: { ibkr_integration: false },
          },
        });

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(
        () => {
          expect(screen.getByTestId('app-version')).toHaveTextContent('1.3.5');
        },
        { timeout: 3000 }
      );
    });

    test('throws error when useApp used outside provider', () => {
      // console.error spy is already created in beforeEach
      expect(() => {
        render(<TestComponent />);
      }).toThrow('useApp must be used within an AppProvider');
    });
  });

  describe('Initial State', () => {
    test('shows connecting message while health check pending', () => {
      api.get.mockImplementation(() => new Promise(() => {})); // Never resolves

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      // Should show the connecting message from AppProvider
      expect(screen.getByText('Connecting to backend...')).toBeInTheDocument();
    });
  });

  describe('Version Info Fetching', () => {
    test('fetches and displays version information', async () => {
      api.get.mockResolvedValueOnce({ data: { status: 'healthy' } }).mockResolvedValueOnce({
        data: {
          app_version: '1.3.5',
          db_version: '1.3.3',
          features: {
            ibkr_integration: false,
            realized_gain_loss: true,
          },
        },
      });

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('app-version')).toHaveTextContent('1.3.5');
        expect(screen.getByTestId('db-version')).toHaveTextContent('1.3.3');
      });
    });

    test('displays feature flags correctly', async () => {
      api.get
        .mockResolvedValueOnce({ data: { status: 'healthy' } })
        .mockResolvedValueOnce({
          data: {
            app_version: '1.3.5',
            db_version: '1.3.3',
            features: {
              ibkr_integration: true,
              realized_gain_loss: true,
            },
          },
        })
        .mockResolvedValueOnce({ data: { configured: false } }); // IBKR config

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('ibkr-feature')).toHaveTextContent('true');
      });
    });
  });

  describe('Error Handling', () => {
    test('handles version fetch error gracefully', async () => {
      api.get
        .mockResolvedValueOnce({ data: { status: 'healthy' } })
        .mockRejectedValueOnce(new Error('Network error'));

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('error')).toHaveTextContent(
          'Failed to load application version information'
        );
      });

      // Should fall back to safe defaults
      expect(screen.getByTestId('app-version')).toHaveTextContent('unknown');
      expect(screen.getByTestId('ibkr-feature')).toHaveTextContent('false');
    });

    test('shows health check error when backend unhealthy', async () => {
      api.get.mockRejectedValueOnce(new Error('Connection refused'));

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('health-check-error')).toBeInTheDocument();
      });
    });

    test('allows retry after health check failure', async () => {
      api.get.mockRejectedValueOnce(new Error('Connection refused'));

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('health-check-error')).toBeInTheDocument();
      });

      // Mock successful retry
      api.get.mockResolvedValueOnce({ data: { status: 'healthy' } }).mockResolvedValueOnce({
        data: {
          app_version: '1.3.5',
          db_version: '1.3.3',
          features: { ibkr_integration: false },
        },
      });

      act(() => {
        screen.getByText('Retry').click();
      });

      await waitFor(() => {
        expect(screen.queryByTestId('health-check-error')).not.toBeInTheDocument();
        expect(screen.getByTestId('app-version')).toHaveTextContent('1.3.5');
      });
    });
  });

  describe('Loading State', () => {
    test('shows loading state while fetching version', async () => {
      api.get.mockResolvedValueOnce({ data: { status: 'healthy' } });

      let resolveVersion;
      api.get.mockImplementationOnce(() => new Promise((resolve) => (resolveVersion = resolve)));

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('true');
      });

      act(() => {
        resolveVersion({
          data: {
            app_version: '1.3.5',
            db_version: '1.3.3',
            features: { ibkr_integration: false },
          },
        });
      });

      await waitFor(() => {
        expect(screen.getByTestId('loading')).toHaveTextContent('false');
      });
    });
  });

  describe('Features Shorthand', () => {
    test('features object is accessible', async () => {
      api.get.mockResolvedValueOnce({ data: { status: 'healthy' } }).mockResolvedValueOnce({
        data: {
          app_version: '1.3.5',
          db_version: '1.3.3',
          features: {
            ibkr_integration: false,
            realized_gain_loss: true,
          },
        },
      });

      render(
        <AppProvider>
          <TestComponent />
        </AppProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('ibkr-feature')).toHaveTextContent('false');
      });
    });
  });
});
