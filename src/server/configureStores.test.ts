jest.mock('ioredis')

import Redis from 'ioredis'
import SocketIO from 'socket.io'
import { configureStores } from './configureStores'
import { MemoryStore, RedisStore } from './store'

describe('configureStores', () => {

  describe('memory', () => {
    it('should be in memory when no params specified', () => {
      const io = SocketIO()
      const stores = configureStores(io)
      expect(stores.socketIdByUserId).toEqual(jasmine.any(MemoryStore))
      expect(stores.userIdBySocketId).toEqual(jasmine.any(MemoryStore))
    })

    it('should be in memory when type="memory"', () => {
      const io = SocketIO()
      const stores = configureStores(io)
      expect(stores.socketIdByUserId).toEqual(jasmine.any(MemoryStore))
      expect(stores.userIdBySocketId).toEqual(jasmine.any(MemoryStore))
    })
  })

  describe('redis', () => {
    it('should be redis when type="redis"', () => {
      const io = SocketIO()
      const stores = configureStores(io, {
        type: 'redis',
        host: 'localhost',
        port: 6379,
        prefix: 'peercalls',
      })
      expect(io.adapter().pubClient).toEqual(jasmine.any(Redis))
      expect(io.adapter().subClient).toEqual(jasmine.any(Redis))
      expect(stores.socketIdByUserId).toEqual(jasmine.any(RedisStore))
      expect(stores.userIdBySocketId).toEqual(jasmine.any(RedisStore))
    })
  })

})
