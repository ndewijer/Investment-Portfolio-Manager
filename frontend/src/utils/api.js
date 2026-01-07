import axios from 'axios';
import { API_BASE_URL } from '../config';

// Enable axios debug mode
axios.interceptors.request.use(
  (request) => {
    console.log('Starting Request:', request);
    return request;
  },
  (error) => {
    // Suppress health check errors
    if (error.config?.url?.includes('system/health')) {
      return Promise.reject(error);
    }
    console.error('Global Request Error:', error);
    return Promise.reject(error);
  }
);

axios.interceptors.response.use(
  (response) => {
    console.log('Response:', response);
    return response;
  },
  (error) => {
    // Suppress health check network errors
    const isHealthCheck = error.config?.url?.includes('system/health');
    const isNetworkError = !error.response && error.request;

    if (isHealthCheck && isNetworkError) {
      return Promise.reject(error);
    }
    console.error('Global Response Error:', error);
    return Promise.reject(error);
  }
);

const api = axios.create({
  baseURL: API_BASE_URL.replace(/\/+$/, ''),
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: false,
});

// Add more detailed logging to the request interceptor
api.interceptors.request.use(
  (config) => {
    console.log('API Request:', {
      url: config.url,
      method: config.method,
      data: config.data,
      headers: config.headers,
    });

    if (config.url) {
      config.url = config.url.replace(/\/+$/, '');
      if (config.url.startsWith('/')) {
        config.url = config.url.substring(1);
      }
    }
    return config;
  },
  (error) => {
    console.error('Request Error:', error);
    return Promise.reject(error);
  }
);

// Add more detailed logging to the response interceptor
api.interceptors.response.use(
  (response) => {
    console.log('API Response:', {
      status: response.status,
      data: response.data,
      headers: response.headers,
    });
    return response;
  },
  (error) => {
    // Suppress error logging for health check endpoint when backend is unavailable
    // This is expected during app initialization when backend is down
    const isHealthCheck = error.config?.url?.includes('system/health');
    const isNetworkError = !error.response && error.request;

    if (isHealthCheck && isNetworkError) {
      // Silently fail health checks - this is expected behavior
      return Promise.reject(error);
    }

    // Log other errors normally
    if (error.response) {
      console.error('Response Error:', {
        status: error.response.status,
        data: error.response.data,
        headers: error.response.headers,
      });
    } else if (error.request) {
      console.error('Request Error:', error.request);
    } else {
      console.error('Error:', error.message);
    }
    return Promise.reject(error);
  }
);

export default api;
