const API_URL = 'http://localhost:5000/api';
// const API_URL = '/api';

export const API_BASE_URL = API_URL;

// Add default axios config for credentials
export const API_CONFIG = {
    withCredentials: true,
    headers: {
        'Content-Type': 'application/json'
    }
};

console.log('API Base URL:', API_BASE_URL);