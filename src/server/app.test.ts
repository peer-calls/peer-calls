jest.mock('socket.io', () => {
  // eslint-disable-next-line
  const { EventEmitter } = require('events')
  return jest.fn().mockReturnValue(new EventEmitter())
})
jest.mock('./socket')

import app from './app'
import { config } from './config'
import handleSocket from './socket'
import SocketIO from 'socket.io'
import request from 'supertest'

const io = SocketIO()

const BASE_URL: string = config.baseUrl

describe('server/app', () => {

  describe('GET /', () => {

    it('renders index', () => {
      return request(app)
      .get('/')
      .expect(200)
    })

  })

  describe('GET /call', () => {

    it('redirects to a new call', () => {
      return request(app)
      .get('/call')
      .expect(302)
      .expect('location', new RegExp(`^${BASE_URL}/call/[0-9a-f-]{36}$`))
    })

  })

  describe('GET /call/<uuid>', () => {

    it('renders call page', () => {
      return request(app)
      .get('/call/test')
      .expect(200)
    })

  })

  describe('io:connection', () => {

    it('calls handleSocket with socket', () => {
      const socket = { hi: 'me socket' }
      io.emit('connection', socket)
      expect((handleSocket as jest.Mock).mock.calls).toEqual([[ socket, io ]])
    })

  })

})
