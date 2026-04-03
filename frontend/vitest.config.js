import { defineConfig } from 'vitest/config';
import { transformWithOxc } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';
import { fileURLToPath } from 'url';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

// Vite 8 uses Rolldown/Oxc which doesn't parse JSX in .js files by default.
// This plugin transforms .js files containing JSX using Oxc.
const transformJsxInJs = () => ({
  name: 'transform-jsx-in-js',
  enforce: 'pre',
  async transform(code, id) {
    if (!id.match(/.*\.js$/)) {
      return null;
    }
    return await transformWithOxc(code, id, {
      lang: 'jsx',
    });
  },
});

export default defineConfig({
  plugins: [react(), transformJsxInJs()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/setupTests.js'],
    include: ['src/**/__tests__/**/*.{js,jsx}', 'src/**/*.{spec,test}.{js,jsx}'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{js,jsx}'],
      exclude: [
        'src/index.js',
        'src/reportWebVitals.js',
        'src/setupTests.js',
        'src/App.js',
        'src/config.js',
        'src/utils/api.js',
        'src/pages/**/*.{js,jsx}',
        'src/components/CollapsibleInfo.js',
        'src/components/FilterPopup.js',
        'src/components/HealthCheckError.js',
        'src/components/Navigation.js',
        'src/components/NumericInput.js',
        'src/components/StatusTab.js',
        'src/components/ValueChart.js',
        'src/components/VersionBanner.js',
        'src/components/portfolio/DividendsTable.js',
        'src/components/portfolio/PortfolioActions.js',
        'src/components/portfolio/TransactionsTable.js',
        'src/context/ThemeContext.js',
        'src/**/*index.js',
        'node_modules/**',
        'src/**/__tests__/**',
      ],
      thresholds: {
        branches: 84,
        functions: 80,
        lines: 90,
        statements: 90,
      },
    },
  },
  resolve: {
    alias: [
      {
        find: /\.(jpg|jpeg|png|gif|svg)$/,
        replacement: resolve(__dirname, '__mocks__/fileMock.js'),
      },
    ],
  },
});
