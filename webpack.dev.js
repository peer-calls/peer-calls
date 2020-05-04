const common = require('./webpack.common.js')
const webpack = require('webpack')
// const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer')

module.exports = {
  ...common,
  devtool: 'inline-source-map',
  plugins: [
    new webpack.SourceMapDevToolPlugin({
      filename: null,
      exclude: [/node_modules/],
      test: /\.tsx?$/,
    }),
    // new BundleAnalyzerPlugin(),
  ],
  mode: 'development',
}
