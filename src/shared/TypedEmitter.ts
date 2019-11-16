type Callback<A> = (a: A) => void

// eslint-disable-next-line
type Events = Record<string | symbol, any>

export type TypedEmitterKeys =
  'removeListener' |
  'on' |
  'once' |
  'off' |
  'emit'

// Some methods might be missing and we do not extend EventEmitter because
// SocketIOClient.Socket does not inherit from EventEmitter, and the method
// signatures differ slightly.
export interface TypedEmitter<E extends Events> {
  removeListener<K extends keyof E>(t: K, callback: Callback<E[K]>): this

  on<K extends keyof E>(t: K, callback: Callback<E[K]>): this
  once<K extends keyof E>(t: K, callback: Callback<E[K]>): this

  emit<K extends keyof E>(t: K, value: E[K]): void
}
