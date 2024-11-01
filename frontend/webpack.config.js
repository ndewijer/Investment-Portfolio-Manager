const path = require('path');

module.exports = {
  mode: process.env.NODE_ENV === 'production' ? 'production' : 'development',
  devtool: process.env.NODE_ENV === 'production' ? 'source-map' : 'eval-source-map',
  output: {
    filename: '[name].bundle.js',
    sourceMapFilename: '[name].js.map',
    path: path.resolve(__dirname, 'build'),
  },
  devServer: {
    historyApiFallback: true,
    hot: true,
    client: {
      overlay: {
        warnings: false,
        errors: true
      }
    }
  }
}; 