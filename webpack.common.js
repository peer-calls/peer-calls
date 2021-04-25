const path = require('path')
const webpack = require('webpack')

const babelLoader = {
  loader: 'babel-loader',
  options: {
    presets: [
      [
        '@babel/preset-react',
      ], [
        '@babel/preset-env',
        {
          'forceAllTransforms': true,
          'targets': {
            'browsers': [
              'last 2 versions',
              'safari >= 7',
              'ie >= 11',
            ],
          },
        },
      ], [
        '@babel/preset-typescript',
      ],
    ],
    plugins: [
      '@babel/plugin-proposal-object-rest-spread',
      '@babel/plugin-proposal-class-properties',
    ],
  },
}

module.exports = {
  entry: {
    index: './src/client/index.tsx',
    'audio.worklet': './src/client/audio/audio.worklet.ts',
  },
  // devtool: 'inline-source-map',
  module: {
    rules: [{
      test: /\.tsx?$/,
      exclude: /node_modules/,
      use: babelLoader,
    }],
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.js', '.mjs'],
  },
  output: {
    filename: '[name].js',
    path: path.resolve(__dirname, 'build'),
  },
  performance: {
    maxEntrypointSize: 650000,
    maxAssetSize: 650000,
  },
  plugins: [
    new webpack.ProvidePlugin({
      process: 'process/browser',
    }),
  ],
  mode: 'development',
}
