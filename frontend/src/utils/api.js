import axios from 'axios';
import { API_BASE_URL } from '../config';

// Remove any trailing slashes from the base URL
const baseURL = API_BASE_URL.replace(/\/+$/, '');

const api = axios.create({
  baseURL,
  headers: {
    'Content-Type': 'application/json',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET,PUT,POST,DELETE,PATCH,OPTIONS',
  },
  withCredentials: false
});

// Add a request interceptor to remove trailing slashes and normalize URLs
api.interceptors.request.use(
  config => {
    // Remove trailing slash from the URL if it exists
    if (config.url) {
      // Remove any trailing slashes from the URL path
      config.url = config.url.replace(/\/+$/, '');
      
      // Ensure there's exactly one slash between baseURL and the path
      if (config.url.startsWith('/')) {
        config.url = config.url.substring(1);
      }
    }
    return config;
  },
  error => {
    return Promise.reject(error);
  }
);

// Add a response interceptor
api.interceptors.response.use(
  response => response,
  error => {
    if (error.response) {
      console.error('Response error:', error.response.data);
    } else if (error.request) {
      console.error('Request error:', error.request);
    } else {
      console.error('Error:', error.message);
    }
    return Promise.reject(error);
  }
);

export default api; 