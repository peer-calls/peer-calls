#!/usr/bin/env node
if (!process.env.DEBUG) {
  process.env.DEBUG = 'peercalls'
}

import _debug from 'debug'
import app from './server/app'

const debug = _debug('peercalls')

const port = parseInt(process.env.PORT || '') || 3000
const hostname = process.env.BIND
const server = app.listen(
  port, hostname, () => debug('Listening on %s', server.address()))


process.on('SIGINT', () => process.exit())
process.on('SIGTERM', () => process.exit())
