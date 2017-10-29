const fs = require('fs')
const path = require('path')

const projectRoot = path.resolve(path.join(__dirname, '../..'))

const readFile = file => fs.readFileSync(path.resolve(projectRoot, file))

function createServer (config, app) {
  if (config.ssl) {
    const key = readFile(config.ssl.key)
    const cert = readFile(config.ssl.cert)
    return require('https').createServer({ key, cert }, app)
  }
  return require('http').createServer(app)
}

module.exports = { createServer }
