#!/usr/bin/env node
'use strict'
if (!process.env.DEBUG) {
  process.env.DEBUG = 'peer-calls:*'
}

const app = require('./server/app.js')
const os = require('os')

let port = process.env.PORT || 3000
let ifaces = os.networkInterfaces()

app.http.listen(port, function () {
  Object.keys(ifaces).forEach(ifname =>
    ifaces[ifname].forEach(iface =>
      console.log('listening on', iface.address, 'and port', port)))
})
