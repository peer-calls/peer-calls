import { Store } from './store'

export class MemoryStore implements Store {
  store: Record<string, string> = {}

  get(key: string): string | undefined {
    return this.store[key]
  }

  set(key: string, value: string) {
    this.store[key] = value
  }

  remove(key: string) {
    delete this.store[key]
  }
}
