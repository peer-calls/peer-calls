import { MemoryStore } from './memory'
import { RedisStore } from './redis'
import Redis from 'ioredis'
import { StoreConfig } from '../config'
import { Store } from './store'

export function createStore(config: StoreConfig = { type: 'memory'}): Store {
  switch (config.type) {
    case 'redis':
      return new RedisStore(new Redis({
        host: config.host,
        port: config.port,
      }), config.prefix)
    default:
      return new MemoryStore()
  }
}
