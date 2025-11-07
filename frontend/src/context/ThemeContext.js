import React, { createContext, useContext, useState, useEffect } from 'react';

const ThemeContext = createContext();

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

// Initialize state from localStorage to avoid setState in effects
const getInitialDarkModeEnabled = () => {
  const savedDarkModeEnabled = localStorage.getItem('darkModeEnabled');
  return savedDarkModeEnabled !== null ? JSON.parse(savedDarkModeEnabled) : false;
};

const getInitialTheme = () => {
  return localStorage.getItem('theme') || 'light';
};

const getInitialSystemPreference = () => {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  return mediaQuery.matches ? 'dark' : 'light';
};

export const ThemeProvider = ({ children }) => {
  // Feature flag to control dark mode availability - load from localStorage
  const [darkModeEnabled, setDarkModeEnabled] = useState(getInitialDarkModeEnabled);

  // Current theme state (light/dark) - load from localStorage
  const [theme, setTheme] = useState(getInitialTheme);

  // Auto-detect system preference - initialize with current media query state
  const [systemPreference, setSystemPreference] = useState(getInitialSystemPreference);

  // Detect system color scheme preference changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

    const handleChange = (e) => {
      setSystemPreference(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement;

    if (darkModeEnabled && theme === 'dark') {
      root.classList.add('dark-theme');
      root.classList.remove('light-theme');
    } else {
      root.classList.add('light-theme');
      root.classList.remove('dark-theme');
    }
  }, [darkModeEnabled, theme]);

  const toggleDarkMode = () => {
    if (!darkModeEnabled) return; // Feature flag check

    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
  };

  const enableDarkModeFeature = (enabled) => {
    setDarkModeEnabled(enabled);
    localStorage.setItem('darkModeEnabled', JSON.stringify(enabled));

    // If disabling dark mode, force light theme
    if (!enabled) {
      setTheme('light');
      localStorage.setItem('theme', 'light');
    }
  };

  const setThemePreference = (newTheme) => {
    if (!darkModeEnabled && newTheme === 'dark') return; // Feature flag check

    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
  };

  const value = {
    // Current state
    theme,
    darkModeEnabled,
    systemPreference,

    // Computed values
    isDark: darkModeEnabled && theme === 'dark',
    isLight: !darkModeEnabled || theme === 'light',

    // Actions
    toggleDarkMode,
    enableDarkModeFeature,
    setThemePreference,
  };

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
};
