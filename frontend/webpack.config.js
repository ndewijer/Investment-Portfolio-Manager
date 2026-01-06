const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const webpack = require('webpack');
const dotenv = require('dotenv');

module.exports = (env, argv) => {
  const isProduction = argv.mode === 'production';

  // Load environment variables
  const envFile = isProduction ? '.env.production' : '.env';
  const envVars = dotenv.config({ path: envFile }).parsed || {};

  // Get domain from environment or build args
  const domain = process.env.DOMAIN || envVars.DOMAIN || 'localhost';
  const useHttps = process.env.USE_HTTPS || envVars.USE_HTTPS || false;

  // Create a default environment if .env file doesn't exist
  const defaultEnv = {
    NODE_ENV: isProduction ? 'production' : 'development',
    DOMAIN: domain,
    USE_HTTPS: useHttps,
    REACT_APP_API_URL: isProduction
      ? `${useHttps ? 'https' : 'http'}://${domain}/api`
      : 'http://localhost:5000/api',
  };

  // Combine default and loaded environment variables
  const combinedEnv = {
    ...defaultEnv,
    ...envVars,
    DOMAIN: domain, // Ensure domain from build args takes precedence
  };

  // Convert environment variables to strings
  const stringifiedEnv = {
    'process.env': Object.keys(combinedEnv).reduce((env, key) => {
      env[key] = JSON.stringify(combinedEnv[key]);
      return env;
    }, {}),
  };

  return {
    entry: './src/index.js',
    output: {
      path: path.resolve(__dirname, 'build'),
      filename: '[name].bundle.js',
      publicPath: '/',
      clean: true,
    },
    module: {
      rules: [
        {
          test: /\.(js|jsx)$/,
          exclude: /node_modules/,
          use: {
            loader: 'babel-loader',
            options: {
              presets: ['@babel/preset-env', '@babel/preset-react'],
              cacheDirectory: true,
            },
          },
        },
        {
          test: /\.css$/,
          use: ['style-loader', 'css-loader'],
        },
        {
          test: /\.(ico|png|jpg|jpeg|gif|svg)$/,
          type: 'asset/resource',
        },
      ],
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: './public/index.html',
        filename: 'index.html',
        favicon: './public/favicon.ico',
        inject: true,
      }),
      new webpack.DefinePlugin(stringifiedEnv),
      // Suppress date-fns locale warning from react-datepicker (dynamic require for locales)
      // This is safe - we only use English locale
      new webpack.IgnorePlugin({
        resourceRegExp: /^\.\/locale$/,
        contextRegExp: /date-fns$/,
      }),
    ],
    resolve: {
      extensions: ['.js', '.jsx'],
    },
    devServer: {
      historyApiFallback: {
        disableDotRule: true,
        rewrites: [
          { from: /^\/favicon.ico$/, to: '/favicon.ico' },
          { from: /^\/[^.]*$/, to: '/index.html' },
        ],
      },
      hot: true,
      port: 3000,
      static: {
        directory: path.join(__dirname, 'public'),
        publicPath: '/',
        serveIndex: false,
      },
      client: {
        overlay: {
          errors: true,
          warnings: false,
        },
      },
      proxy: [
        {
          context: ['/api'],
          target: 'http://localhost:5000',
          secure: false,
          pathRewrite: { '^/api': '/api' },
          changeOrigin: true,
        },
      ],
    },
    devtool: isProduction ? false : 'eval-source-map', // Disable source maps in production
    performance: {
      maxAssetSize: 1000000, // 1MB per asset
      maxEntrypointSize: 1700000, // 1.7MB total entrypoint (with some buffer)
      hints: 'warning', // Warn but don't fail build - bundle analyzer available for debugging
    },
    ignoreWarnings: [
      // Suppress react-datepicker's dynamic locale loading warning (harmless)
      /Critical dependency: the request of a dependency is an expression/,
      /react-datepicker/,
    ],
    optimization: {
      moduleIds: 'deterministic', // Better for caching
      minimize: isProduction,
      splitChunks: {
        chunks: 'all',
        cacheGroups: {
          // Separate vendor bundle for better caching
          vendors: {
            test: /[\\/]node_modules[\\/]/,
            priority: -10,
            reuseExistingChunk: true,
          },
          // Separate bundle for react-datepicker and date-fns (if used)
          datepicker: {
            test: /[\\/]node_modules[\\/](react-datepicker|date-fns)[\\/]/,
            name: 'datepicker',
            priority: 10,
            reuseExistingChunk: true,
          },
          default: {
            minChunks: 2,
            priority: -20,
            reuseExistingChunk: true,
          },
        },
      },
    },
  };
};
