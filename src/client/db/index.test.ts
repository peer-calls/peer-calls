import { open, promisify } from './index'

describe('db', () => {

  const TEST_DB = 'TEST_DB'

  async function getError(promise: Promise<unknown>): Promise<Error> {
    let error: Error
    try {
      await promise
    } catch (err) {
      error = err
    }
    expect(error!).toBeTruthy()
    return error!
  }

  afterEach(async () => {
    db && db.close()
    await promisify(window.indexedDB.deleteDatabase(TEST_DB))
  })
  let db: IDBDatabase

  describe('open', () => {

    it('can use a custom upgrade function', async () => {
      let called = false
      db = await open(TEST_DB, 1, ev => {
        called = true
      })
      expect(called).toBe(true)
    })

    it('opens a new database and upgrades it', async () => {
      db = await open(TEST_DB, 1)
      const tx = db.transaction('identities', 'readwrite')
      const store = tx.objectStore('identities')
      await promisify(store.put({id: 'test'}))
      const value = await promisify(store.get('test'))
      expect(value).toEqual({id: 'test'})
      await promisify(tx)
    })

  })

})
