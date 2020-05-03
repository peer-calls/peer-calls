const path = require('path')

module.exports = {
  entry: './src/client/index.tsx',
  // devtool: 'inline-source-map',
  module: {
    rules: [{
      // test: /\.tsx?$/,
      // use: 'ts-loader',
      // exclude: /node_modules/,
    // }, {
      test: /\.tsx?$/,
      exclude: /node_modules/,
      use: {
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
      },
    }],
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.js', '.mjs'],
  },
  output: {
    filename: 'index.js',
    path: path.resolve(__dirname, 'build'),
  },
  performance: {
    maxEntrypointSize: 650000,
    maxAssetSize: 650000,
  },
  mode: 'development',
}
