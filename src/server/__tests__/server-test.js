const express = require('express')
const http = require('http')
const https = require('https')
const { createServer } = require('../server.js')

describe('server', () => {

  let app, config
  beforeEach(() => {
    config = {}
    app = express()
  })

  describe('createServer', () => {
    it('creates https server when config.ssl', () => {
      config.ssl = {
        cert: 'config/cert.example.pem',
        key: 'config/cert.example.key'
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
