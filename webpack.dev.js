const common = require('./webpack.common.js')
// const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer')

module.exports = {
  ...common,
  // plugins: [
  //   new BundleAnalyzerPlugin(),
  // ],
  mode: 'development',
}
