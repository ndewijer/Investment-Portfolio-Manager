import React, { createContext, useContext, useState, useEffect } from 'react';

const ThemeContext = createContext();

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

export const ThemeProvider = ({ children }) => {
  // Feature flag to control dark mode availability
  const [darkModeEnabled, setDarkModeEnabled] = useState(false);

  // Current theme state (light/dark)
  const [theme, setTheme] = useState('light');

  // Auto-detect system preference
  const [systemPreference, setSystemPreference] = useState('light');

  // Detect system color scheme preference
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    setSystemPreference(mediaQuery.matches ? 'dark' : 'light');

    const handleChange = (e) => {
      setSystemPreference(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  // Load saved preferences from localStorage
  useEffect(() => {
    const savedDarkModeEnabled = localStorage.getItem('darkModeEnabled');
    const savedTheme = localStorage.getItem('theme');

    if (savedDarkModeEnabled !== null) {
      setDarkModeEnabled(JSON.parse(savedDarkModeEnabled));
    }

    if (savedTheme) {
      setTheme(savedTheme);
    }
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
