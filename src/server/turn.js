'use strict'
const crypto = require('crypto')

function getCredentials (name, secret) {
  // this credential would be valid for the next 24 hours
  const timestamp = parseInt(Date.now() / 1000, 10) + 24 * 3600
  const username = [timestamp, name].join(':')
  const hmac = crypto.createHmac('sha1', secret)
  hmac.setEncoding('base64')
  hmac.write(username)
  hmac.end()
  const credential = hmac.read()
  return { username, credential }
}

function processServers (iceServers) {
  return iceServers.map(server => {
    switch (server.auth) {
      case undefined:
        return server
      case 'secret':
        const cred = getCredentials(server.username, server.secret)
        return {
          url: server.url,
          urls: server.urls,
          username: cred.username,
          credential: cred.credential
        }
      default:
        throw new Error('Authentication type not implemented: ' + server.auth)
    }
  })
}

module.exports = { getCredentials, processServers }
