const Encore = require('@symfony/webpack-encore');

Encore
  .setOutputPath('build/')
  .setPublicPath('/')
  .cleanupOutputBeforeBuild()
  .enableBuildNotifications()
  .enableSourceMaps(!Encore.isProduction())
  .enableVersioning(Encore.isProduction())
  .addEntry('js/index', './src/client/index.js')
  .addStyleEntry('css/style', './src/scss/style.scss')
  .enableSassLoader()
  .enableReactPreset()
  .disableSingleRuntimeChunk()
  .autoProvidejQuery()
  .configureDefinePlugin((options) => {
    options['process.env.APP_GOOGLE_MAPS_API_KEY'] =
      JSON.stringify(process.env.APP_GOOGLE_MAPS_API_KEY)
  })
  .copyFiles({
    from: './src/assets/images',
    to: 'images/[path][name].[ext]'
  })
;

module.exports = Encore.getWebpackConfig();
