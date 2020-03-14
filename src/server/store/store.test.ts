import { MemoryStore, RedisStore } from './'
import Redis from 'ioredis'
import { Store } from './store'

describe('store', () => {

  const redis = new Redis({
    host: process.env.TEST_REDIS_HOST || 'localhost',
    port: parseInt(process.env.TEST_REDIS_PORT!) || 6379,
    enableOfflineQueue: true,
  })

  const testCases: Array<{name: string, store: Store}> = [{
    name: MemoryStore.name,
    store: new MemoryStore(),
  }, {
    name: RedisStore.name,
    store: new RedisStore(redis, 'peercallstest'),
  }]

  afterAll(() => {
    redis.disconnect()
  })

  testCases.forEach(({name, store}) => {
    describe(name, () => {
      afterEach(async () => {
        await Promise.all([
          store.remove('a'),
          store.remove('b'),
        ])
      })
      describe('set, get, getMany', () => {
        it('sets and retreives value(s)', async () => {
          await store.set('a', 'A')
          await store.set('b', 'B')
          expect(await store.get('a')).toBe('A')
          expect(await store.get('b')).toBe('B')
          expect(await store.remove('b'))
          expect(await store.get('c')).toBe(undefined)
          expect(await store.getMany(['a', 'b', 'c']))
          .toEqual(['A', undefined, undefined])
        })
      })
    })
  })
})
