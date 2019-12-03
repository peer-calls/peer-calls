export type Upgrade =
  (this: IDBOpenDBRequest, event: IDBVersionChangeEvent) => void

export function upgrade(this: IDBOpenDBRequest, event: IDBVersionChangeEvent) {
  const db = this.result
  switch (event.oldVersion) {
    case 0:
      db.createObjectStore('identities', {
        keyPath: 'id',
      })
  }
}
