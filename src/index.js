#!/usr/bin/env node
'use strict'
if (!process.env.DEBUG) {
  process.env.DEBUG = 'peercalls'
}

const app = require('./server/app.js')
const debug = require('debug')('peercalls')

let port = process.env.PORT || 3000
app.http.listen(port, () => debug('Listening on: %s', port))
