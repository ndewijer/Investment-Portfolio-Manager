// Get the domain from webpack-defined environment or use default
const DOMAIN = process.env.DOMAIN || 'localhost';
const IS_PRODUCTION = process.env.NODE_ENV === 'production';

// In development, use localhost, in production use the domain
const API_URL = IS_PRODUCTION 
  ? `https://${DOMAIN}/api`
  : 'http://localhost:5000/api';

export const API_BASE_URL = API_URL;

// Add default axios config for credentials
export const API_CONFIG = {
    withCredentials: true,
    headers: {
        'Content-Type': 'application/json'
    }
};

console.log('Environment:', process.env.NODE_ENV);
console.log('Domain:', DOMAIN);
console.log('API Base URL:', API_BASE_URL);