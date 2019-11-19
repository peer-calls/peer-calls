import express from 'express'
import http, { RequestListener } from 'http'
import https from 'https'
import { createServer } from './server'

describe('server', () => {

  let app: RequestListener
  let config: any
  beforeEach(() => {
    config = {}
    app = express()
  })

  describe('createServer', () => {
    it('creates https server when config.ssl', () => {
      config.ssl = {
        cert: 'config/cert.example.pem',
        key: 'config/cert.example.key',
      }
      const s = createServer(config, app)
      expect(s).toEqual(jasmine.any(https.Server))
    })
    it('creates http server when no ssl config', () => {
      const s = createServer(config, app)
      expect(s).toEqual(jasmine.any(http.Server))
    })
  })

})
