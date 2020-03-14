import { Store } from './store'

export class MemoryStore implements Store {
  store: Record<string, string> = {}

  async getMany(keys: string[]): Promise<Array<string | undefined>> {
    return keys.map(key => this.syncGet(key))
  }

  private syncGet(key: string): string | undefined {
    return  this.store[key]
  }

  async get(key: string): Promise<string | undefined> {
    return this.syncGet(key)
  }

  async set(key: string, value: string) {
    this.store[key] = value
  }

  async remove(key: string) {
    delete this.store[key]
  }
}
