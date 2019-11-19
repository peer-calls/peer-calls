#!/usr/bin/env node
if (!process.env.DEBUG) {
  process.env.DEBUG = 'peercalls'
}

import _debug from 'debug'
import app from './server/app'

const debug = _debug('peercalls')

const port = process.env.PORT || 3000
app.listen(port, () => debug('Listening on: %s', port))

process.on('SIGINT', () => process.exit())
process.on('SIGTERM', () => process.exit())
