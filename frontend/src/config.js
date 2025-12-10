// Determine if we're in development mode (npm start)
const IS_DEV = process.env.NODE_ENV === 'development';

// In development, connect directly to backend on localhost:5000
// In production (Docker), use relative URL - nginx will proxy to backend
const API_URL = IS_DEV ? 'http://localhost:5000/api' : '/api';

export const API_BASE_URL = API_URL;

// Add default axios config for credentials
export const API_CONFIG = {
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
};

// Log configuration for debugging
console.log('Config environment:', {
  IS_DEV,
  API_BASE_URL,
});
