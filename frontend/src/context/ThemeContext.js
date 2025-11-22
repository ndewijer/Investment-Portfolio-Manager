import React, { createContext, useContext, useState, useEffect } from 'react';

/**
 * Theme context that provides dark mode support and theme management.
 * Controls both the feature flag for dark mode and the current theme preference.
 *
 * @context ThemeContext
 * @see ThemeProvider
 * @see useTheme
 */
const ThemeContext = createContext();

/**
 * Hook to access the theme context.
 * Provides access to current theme state, system preference, and theme management functions.
 *
 * @returns {Object} The theme context value
 * @returns {string} returns.theme - Current theme setting ('light' or 'dark')
 * @returns {boolean} returns.darkModeEnabled - Whether dark mode feature is enabled
 * @returns {string} returns.systemPreference - System color scheme preference ('light' or 'dark')
 * @returns {boolean} returns.isDark - Whether dark mode is currently active
 * @returns {boolean} returns.isLight - Whether light mode is currently active
 * @returns {Function} returns.toggleDarkMode - Function to toggle between light and dark themes
 * @returns {Function} returns.enableDarkModeFeature - Function to enable/disable dark mode feature
 * @returns {Function} returns.setThemePreference - Function to set a specific theme preference
 *
 * @throws {Error} If used outside of ThemeProvider
 *
 * @example
 * const { theme, isDark, toggleDarkMode } = useTheme();
 *
 * @example
 * // Toggle dark mode
 * const { toggleDarkMode, darkModeEnabled } = useTheme();
 * if (darkModeEnabled) {
 *   toggleDarkMode();
 * }
 */
export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

/**
 * Retrieves the initial dark mode enabled state from localStorage.
 * Defaults to false if no saved preference exists.
 *
 * @function getInitialDarkModeEnabled
 * @returns {boolean} The saved dark mode enabled state or false
 */
const getInitialDarkModeEnabled = () => {
  const savedDarkModeEnabled = localStorage.getItem('darkModeEnabled');
  return savedDarkModeEnabled !== null ? JSON.parse(savedDarkModeEnabled) : false;
};

/**
 * Retrieves the initial theme preference from localStorage.
 * Defaults to 'light' if no saved preference exists.
 *
 * @function getInitialTheme
 * @returns {string} The saved theme preference ('light' or 'dark') or 'light'
 */
const getInitialTheme = () => {
  return localStorage.getItem('theme') || 'light';
};

/**
 * Retrieves the system's color scheme preference.
 * Checks the prefers-color-scheme media query.
 *
 * @function getInitialSystemPreference
 * @returns {string} The system preference ('light' or 'dark')
 */
const getInitialSystemPreference = () => {
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  return mediaQuery.matches ? 'dark' : 'light';
};

/**
 * Provider component for theme context.
 * Manages theme state, dark mode feature flag, and system preference detection.
 *
 * This provider:
 * - Initializes theme state from localStorage
 * - Monitors system color scheme preference changes
 * - Applies theme classes to the document root
 * - Provides methods to toggle and manage theme preferences
 *
 * @param {Object} props - Component props
 * @param {React.ReactNode} props.children - Child components that will have access to the context
 * @returns {JSX.Element} Provider component
 *
 * @example
 * <ThemeProvider>
 *   <App />
 * </ThemeProvider>
 */
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

  /**
   * Toggles between light and dark themes.
   * Only works if dark mode feature is enabled. Updates both state and localStorage.
   *
   * @function toggleDarkMode
   * @returns {void}
   */
  const toggleDarkMode = () => {
    if (!darkModeEnabled) return; // Feature flag check

    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
  };

  /**
   * Enables or disables the dark mode feature.
   * When disabled, forces the theme to light mode. Updates both state and localStorage.
   *
   * @function enableDarkModeFeature
   * @param {boolean} enabled - Whether to enable dark mode feature
   * @returns {void}
   */
  const enableDarkModeFeature = (enabled) => {
    setDarkModeEnabled(enabled);
    localStorage.setItem('darkModeEnabled', JSON.stringify(enabled));

    // If disabling dark mode, force light theme
    if (!enabled) {
      setTheme('light');
      localStorage.setItem('theme', 'light');
    }
  };

  /**
   * Sets a specific theme preference.
   * Respects dark mode feature flag - prevents setting dark theme if feature is disabled.
   * Updates both state and localStorage.
   *
   * @function setThemePreference
   * @param {string} newTheme - The theme to set ('light' or 'dark')
   * @returns {void}
   */
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
