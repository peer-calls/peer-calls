export interface Store {
  set(key: string, value: string): Promise<void>
  get(key: string): Promise<string | undefined>
  getMany(keys: string[]): Promise<Array<string | undefined>>
  remove(key: string): Promise<void>
}
