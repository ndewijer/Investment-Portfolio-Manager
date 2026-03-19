import React, { createContext, useContext, useEffect, useState } from 'react';

/**
 * Theme context that provides dark mode support and theme management.
 * Manages the current theme preference (light or dark).
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
 * @returns {string} returns.systemPreference - System color scheme preference ('light' or 'dark')
 * @returns {boolean} returns.isDark - Whether dark mode is currently active
 * @returns {boolean} returns.isLight - Whether light mode is currently active
 * @returns {Function} returns.toggleDarkMode - Function to toggle between light and dark themes
 * @returns {Function} returns.setThemePreference - Function to set a specific theme preference
 *
 * @throws {Error} If used outside of ThemeProvider
 *
 * @example
 * const { theme, isDark, toggleDarkMode } = useTheme();
 */
export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};

/**
 * Retrieves the initial theme preference from localStorage.
 * Falls back to system preference if no saved preference exists.
 *
 * @function getInitialTheme
 * @returns {string} The saved theme preference ('light' or 'dark'), or system preference
 */
const getInitialTheme = () => {
  const saved = localStorage.getItem('theme');
  if (saved) return saved;

  // Fall back to system preference on first load
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  return mediaQuery.matches ? 'dark' : 'light';
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
 * Manages theme state and system preference detection.
 *
 * This provider:
 * - Initializes theme state from localStorage (falls back to system preference)
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
  // Current theme state (light/dark) - load from localStorage or system preference
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

  // Apply theme to document - uses data-theme attribute for 2026 design system
  useEffect(() => {
    const root = document.documentElement;

    if (theme === 'dark') {
      root.setAttribute('data-theme', 'dark');
      root.classList.add('dark-theme');
      root.classList.remove('light-theme');
    } else {
      root.removeAttribute('data-theme');
      root.classList.add('light-theme');
      root.classList.remove('dark-theme');
    }
  }, [theme]);

  /**
   * Toggles between light and dark themes.
   * Updates both state and localStorage.
   *
   * @function toggleDarkMode
   * @returns {void}
   */
  const toggleDarkMode = () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
  };

  /**
   * Sets a specific theme preference.
   * Updates both state and localStorage.
   *
   * @function setThemePreference
   * @param {string} newTheme - The theme to set ('light' or 'dark')
   * @returns {void}
   */
  const setThemePreference = (newTheme) => {
    setTheme(newTheme);
    localStorage.setItem('theme', newTheme);
  };

  const value = {
    // Current state
    theme,
    systemPreference,

    // Computed values
    isDark: theme === 'dark',
    isLight: theme === 'light',

    // Actions
    toggleDarkMode,
    setThemePreference,
  };

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
};
