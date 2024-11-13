// Get the domain from webpack-defined environment or use default
const DOMAIN = process.env.DOMAIN || 'localhost';
const USE_HTTPS = process.env.USE_HTTPS === 'true';
const IS_PRODUCTION = process.env.NODE_ENV === 'production';

// Determine protocol based on USE_HTTPS
const PROTOCOL = USE_HTTPS ? 'https' : 'http';

// Determine API URL based on environment
const API_URL = IS_PRODUCTION ? `${PROTOCOL}://${DOMAIN}/api` : 'http://localhost:5000/api';

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
  DOMAIN,
  USE_HTTPS,
  IS_PRODUCTION,
  API_BASE_URL,
});
