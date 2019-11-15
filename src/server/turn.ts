import crypto from 'crypto'
import { ICEServer } from './config'

export interface Credentials {
  username: string
  credential: string
}

export function getCredentials (name: string, secret: string): Credentials {
  // this credential would be valid for the next 24 hours
  const timestamp = Math.floor(Date.now() / 1000) + 24 * 3600
  const username = [timestamp, name].join(':')
  const hmac = crypto.createHmac('sha1', secret)
  hmac.setEncoding('base64')
  hmac.write(username)
  hmac.end()
  const credential = hmac.read()
  return { username, credential }
}

function getServerConfig(server: ICEServer, cred: Credentials) {
  return {
    url: server.url,
    urls: server.urls,
    username: cred.username,
    credential: cred.credential,
  }
}

export function processServers (iceServers: ICEServer[]) {
  return iceServers.map(server => {
    switch (server.auth) {
      case undefined:
        return server
      case 'secret':
        return getServerConfig(
          server,
          getCredentials(server.username, server.secret),
        )
      default:
        throw new Error('Authentication type not implemented: ' +
                        (server as {auth: string}).auth)
    }
  })
}
