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

  // Create a default environment if .env file doesn't exist
  const defaultEnv = {
    NODE_ENV: isProduction ? 'production' : 'development',
    DOMAIN: domain,
    REACT_APP_API_URL: isProduction ? `https://${domain}/api` : 'http://localhost:5000/api',
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
            },
          },
        },
        {
          test: /\.css$/,
          use: ['style-loader', 'css-loader'],
        },
      ],
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: './public/index.html',
        filename: 'index.html',
        inject: true,
      }),
      new webpack.DefinePlugin(stringifiedEnv),
    ],
    resolve: {
      extensions: ['.js', '.jsx'],
    },
    devServer: {
      historyApiFallback: true,
      hot: true,
      port: 3000,
      static: {
        directory: path.join(__dirname, 'public'),
        publicPath: '/',
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
    devtool: isProduction ? 'source-map' : 'eval-source-map',
    performance: {
      hints: false,
    },
  };
};
