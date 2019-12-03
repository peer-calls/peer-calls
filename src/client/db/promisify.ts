export async function promisify(request: IDBTransaction): Promise<void>
export async function promisify<T>(request: IDBRequest<T>): Promise<T>
export async function promisify<T>(request: IDBRequest<T> | IDBTransaction) {
  if ('oncomplete' in request) {
    // this is a transaction
    return new Promise<void>((resolve, reject) => {
      request.oncomplete = () => resolve()
      request.onerror = err => reject(err)
    })
  }
  // this is an IDBRequest
  return new Promise<T>((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = err => reject(err)
  })
}
