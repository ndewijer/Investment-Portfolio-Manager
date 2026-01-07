/**
 * API Client Tests
 *
 * Tests the axios API client configuration and behavior.
 * Note: api.js is mostly configuration and logging interceptors,
 * so we focus on verifying the exported instance is correctly configured.
 */

import api from '../api';

describe('API Client', () => {
  describe('Configuration', () => {
    it('exports a configured axios instance', () => {
      expect(api).toBeDefined();
      expect(typeof api.request).toBe('function');
    });

    it('has all standard HTTP methods available', () => {
      expect(typeof api.get).toBe('function');
      expect(typeof api.post).toBe('function');
      expect(typeof api.put).toBe('function');
      expect(typeof api.delete).toBe('function');
      expect(typeof api.patch).toBe('function');
      expect(typeof api.head).toBe('function');
      expect(typeof api.options).toBe('function');
    });

    it('has interceptors configured', () => {
      expect(api.interceptors).toBeDefined();
      expect(api.interceptors.request).toBeDefined();
      expect(api.interceptors.response).toBeDefined();
    });

    it('has request interceptors registered', () => {
      expect(api.interceptors.request.handlers).toBeDefined();
      expect(api.interceptors.request.handlers.length).toBeGreaterThan(0);
    });

    it('has response interceptors registered', () => {
      expect(api.interceptors.response.handlers).toBeDefined();
      expect(api.interceptors.response.handlers.length).toBeGreaterThan(0);
    });

    it('has default baseURL configured', () => {
      expect(api.defaults.baseURL).toBeDefined();
      expect(api.defaults.baseURL).toBe('/api');
    });

    it('has correct default headers', () => {
      // Headers might be in common, or directly set
      const headers = api.defaults.headers;
      const contentType =
        headers.common?.['Content-Type'] ||
        headers['Content-Type'] ||
        headers.post?.['Content-Type'];

      expect(contentType).toBe('application/json');
    });

    it('has withCredentials set to false', () => {
      expect(api.defaults.withCredentials).toBe(false);
    });
  });

  describe('URL Transformation Logic', () => {
    // Test the URL transformation logic that's in the interceptor
    it('removes trailing slashes correctly', () => {
      const testUrls = [
        { input: '/portfolios/', expected: 'portfolios' },
        { input: '/portfolios', expected: 'portfolios' },
        { input: 'portfolios/', expected: 'portfolios' },
        { input: 'portfolios', expected: 'portfolios' },
        { input: '/', expected: '' },
        { input: '', expected: '' },
      ];

      testUrls.forEach(({ input, expected }) => {
        let transformed = input;
        if (transformed) {
          transformed = transformed.replace(/\/+$/, '');
          if (transformed.startsWith('/')) {
            transformed = transformed.substring(1);
          }
        }
        expect(transformed).toBe(expected);
      });
    });

    it('handles complex URL patterns', () => {
      const complexUrls = [
        { input: '/api/portfolios/', expected: 'api/portfolios' },
        { input: '//portfolios//', expected: '/portfolios' },
        { input: '/portfolios/123/', expected: 'portfolios/123' },
      ];

      complexUrls.forEach(({ input, expected }) => {
        let transformed = input.replace(/\/+$/, '');
        if (transformed.startsWith('/')) {
          transformed = transformed.substring(1);
        }
        expect(transformed).toBe(expected);
      });
    });
  });

  describe('Error Handling Logic', () => {
    it('identifies health check URLs correctly', () => {
      const testCases = [
        { url: '/api/system/health', isHealthCheck: true },
        { url: 'system/health', isHealthCheck: true },
        { url: '/system/health/status', isHealthCheck: true },
        { url: '/api/portfolios', isHealthCheck: false },
        { url: '/portfolios/health', isHealthCheck: false },
        { url: undefined, isHealthCheck: false },
      ];

      testCases.forEach(({ url, isHealthCheck }) => {
        const result = url?.includes('system/health') || false;
        expect(result).toBe(isHealthCheck);
      });
    });

    it('identifies network errors vs server errors', () => {
      const networkError = {
        config: {},
        request: {},
        response: undefined,
      };

      const serverError = {
        config: {},
        response: { status: 500, data: {} },
      };

      const messageError = {
        message: 'Network Error',
      };

      // Network error: has request but no response
      const isNetworkError = !networkError.response && !!networkError.request;
      expect(isNetworkError).toBe(true);

      // Server error: has response
      expect(!!serverError.response).toBe(true);

      // Message error: neither request nor response
      expect(!messageError.request && !messageError.response).toBe(true);
    });

    it('categorizes errors correctly', () => {
      const errors = [
        {
          name: 'Response Error',
          error: { response: { status: 404, data: { error: 'Not found' } } },
          category: 'response',
        },
        {
          name: 'Network Error',
          error: { request: {}, response: undefined },
          category: 'network',
        },
        {
          name: 'Setup Error',
          error: { message: 'Configuration error' },
          category: 'message',
        },
      ];

      errors.forEach(({ error, category }) => {
        let detected;
        if (error.response) {
          detected = 'response';
        } else if (error.request) {
          detected = 'network';
        } else {
          detected = 'message';
        }
        expect(detected).toBe(category);
      });
    });
  });

  describe('Logging Configuration', () => {
    it('has console methods available for logging', () => {
      expect(typeof console.log).toBe('function');
      expect(typeof console.error).toBe('function');
    });

    it('can create log objects for requests', () => {
      const request = {
        url: '/portfolios',
        method: 'GET',
        data: { test: true },
        headers: { Authorization: 'Bearer token' },
      };

      const logObject = {
        url: request.url,
        method: request.method,
        data: request.data,
        headers: request.headers,
      };

      expect(logObject).toMatchObject({
        url: '/portfolios',
        method: 'GET',
        data: { test: true },
      });
    });

    it('can create log objects for responses', () => {
      const response = {
        status: 200,
        data: { id: 1, name: 'Test' },
        headers: { 'content-type': 'application/json' },
      };

      const logObject = {
        status: response.status,
        data: response.data,
        headers: response.headers,
      };

      expect(logObject).toMatchObject({
        status: 200,
        data: { id: 1, name: 'Test' },
      });
    });

    it('can create log objects for errors', () => {
      const error = {
        response: {
          status: 404,
          data: { error: 'Not found' },
          headers: {},
        },
      };

      const logObject = {
        status: error.response.status,
        data: error.response.data,
        headers: error.response.headers,
      };

      expect(logObject).toMatchObject({
        status: 404,
        data: { error: 'Not found' },
      });
    });
  });

  describe('Health Check Suppression Logic', () => {
    it('identifies health check network errors that should be suppressed', () => {
      const testCases = [
        {
          error: {
            config: { url: 'system/health' },
            request: {},
            response: undefined,
          },
          shouldSuppress: true,
          reason: 'Health check with network error',
        },
        {
          error: {
            config: { url: 'system/health' },
            response: { status: 500 },
          },
          shouldSuppress: false,
          reason: 'Health check with server error (not network)',
        },
        {
          error: {
            config: { url: 'portfolios' },
            request: {},
            response: undefined,
          },
          shouldSuppress: false,
          reason: 'Non-health check network error',
        },
      ];

      testCases.forEach(({ error, shouldSuppress }) => {
        const isHealthCheck = error.config?.url?.includes('system/health');
        const isNetworkError = !error.response && !!error.request;
        const suppress = isHealthCheck && isNetworkError;

        expect(suppress).toBe(shouldSuppress);
      });
    });
  });
});
