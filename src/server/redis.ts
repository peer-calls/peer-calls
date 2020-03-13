import _debug from 'debug'
import Redis from 'ioredis'
import redisAdapter from 'socket.io-redis'
import { StoreConfig, StoreRedisConfig } from './config'
import { Stores } from './socket'
import { MemoryStore, RedisStore } from './store'

const debug = _debug('peercalls')

export function configureStores(
  io: SocketIO.Server,
  config: StoreConfig = { type: 'memory'},
): Stores {
  switch (config.type) {
    case 'redis':
      debug('Using redis store: %s:%s', config.host, config.port)
      configureRedis(io, config)
      return {
        socketIdByUserId: new RedisStore(
          createRedisClient(config),
          [config.prefix, 'socketIdByUserId'].join(':'),
        ),
        userIdBySocketId: new RedisStore(
          createRedisClient(config),
          [config.prefix, 'socketIdByUserId'].join(':'),
        ),
      }
    default:
      debug('Using in-memory store')
      return {
        socketIdByUserId: new MemoryStore(),
        userIdBySocketId: new MemoryStore(),
      }
  }
}

function configureRedis(
  io: SocketIO.Server,
  config: StoreRedisConfig,
) {
  const pubClient = createRedisClient(config)
  const subClient = createRedisClient(config)
  io.adapter(redisAdapter({
    key: 'peercalls',
    pubClient,
    subClient,
  }))
}

function createRedisClient(config: StoreRedisConfig) {
  return new Redis({
    host: config.host,
    port: config.port,
  })
}
