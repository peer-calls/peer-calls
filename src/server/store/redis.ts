import Redis from 'ioredis'
import { Store } from './store'
import _debug from 'debug'

const debug = _debug('peercalls:redis')

interface RedisClient {
  get: Redis.Redis['get']
  set: Redis.Redis['set']
  del: Redis.Redis['del']
}
export class RedisStore implements Store {
  constructor(
    protected readonly redis: RedisClient,
    protected readonly prefix: string,
  ) {
  }

  private getKey(key: string): string {
    return [this.prefix, key].filter(k => !!k).join(':')
  }

  private nullToUndefined(value: string | null): string | undefined {
    if (value === null) {
      return undefined
    }
    return value
  }

  async getMany(keys: string[]): Promise<Array<string | undefined>> {
    const result = await Promise.all(
      keys.map(key => this.redis.get(this.getKey(key))))
    return result.map(this.nullToUndefined)
  }

  async get(key: string): Promise<string | undefined> {
    key = this.getKey(key)
    debug('get %s', key)
    const result = await this.redis.get(key)
    return this.nullToUndefined(result)
  }

  async set(key: string, value: string) {
    key = this.getKey(key)
    debug('set %s %s', key, value)
    await this.redis.set(key, value)
  }

  async remove(key: string) {
    key = this.getKey(key)
    debug('del %s', key)
    await this.redis.del(key)
  }
}
