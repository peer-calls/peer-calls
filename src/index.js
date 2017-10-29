#!/usr/bin/env node
'use strict'
if (!process.env.DEBUG) {
  process.env.DEBUG = 'peercalls'
}

const app = require('./server/app.js')
const debug = require('debug')('peercalls')

const port = process.env.PORT || 3000
const server = app.listen(port, () => debug('Listening on: %s', port))

function close () {
  debug('Closing server...')
  server.close(() => {
    debug('Bye!')
    process.exit()
  })
}

process.on('SIGINT', close)
process.on('SIGTERM', close)
