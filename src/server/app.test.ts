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
import { MemoryStore } from './store'

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

  describe('POST /call', () => {

    it('redirects to a new call', () => {
      return request(app)
      .post('/call')
      .expect(302)
      .expect('location', new RegExp(`^${BASE_URL}/call/[0-9a-f-]{36}$`))
    })

    it('redirects to specific call', () => {
      return request(app)
      .post('/call')
      .send('call=test%20id')
      .expect(302)
      .expect('location', `${BASE_URL}/call/test%20id`)
    })

  })

  describe('GET /call/<uuid>', () => {

    it('renders call page', () => {
      return request(app)
      .get(`${BASE_URL}/call/test`)
      .expect(200)
    })

    it('sets nickname from x-forwarded-user', () => {
      return request(app)
      .get(`${BASE_URL}/call/test`)
      .set('x-forwarded-user', 'abc')
      .expect(200)
      .expect(/<input type="hidden" id="nickname" value="abc">/)
    })

  })

  describe('io:connection', () => {

    it('calls handleSocket with socket', () => {
      const socket = { hi: 'me socket' }
      io.emit('connection', socket)
      expect((handleSocket as jest.Mock).mock.calls).toEqual([[
        socket,
        io,
        {
          socketIdByUserId: jasmine.any(MemoryStore),
          userIdBySocketId: jasmine.any(MemoryStore),
        },
      ]])
    })

  })

})
