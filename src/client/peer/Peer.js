'use strict'
const Peer = require('simple-peer')

function init (opts) {
  return Peer(opts)
}

module.exports = { init }
