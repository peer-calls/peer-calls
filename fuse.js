const { fusebox, pluginReplace } = require('fuse-box')

fusebox({
  target: 'browser',
  entry: 'src/client/index.tsx',
  cache : true,
  devServer: false,
  watcher: {
    enabled: true,
    include: ['src/'],
    ignore: ['build/'],
  },
  plugins: [
    pluginReplace(/node_modules\/readable-stream\/.*/, {
      'require(\'util\')':
        'require(\'' + require.resolve('./node_modules/util') + '\')',
    }),
  ],
})
.runDev({
  bundles: {
    distRoot: '.',
    app: 'build/index.js',
  },
})
