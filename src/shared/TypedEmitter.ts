import  { EventEmitter } from 'events'

type Callback<A> = (a: A) => void

// eslint-disable-next-line
type Events = Record<string | symbol, any>

export type TypedEmitterKeys =
  'addListener' |
  'removeListener' |
  'on' |
  'once' |
  'off' |
  'emit'

export interface TypedEmitter<E extends Events>
extends EventEmitter {
  addListener<K extends keyof E>(t: K, callback: Callback<E[K]>): this
  removeListener<K extends keyof E>(t: K, callback: Callback<E[K]>): this

  on<K extends keyof E>(t: K, callback: Callback<E[K]>): this
  once<K extends keyof E>(t: K, callback: Callback<E[K]>): this

  off<K extends keyof E>(t: K, callback: Callback<E[K]>): this

  emit<K extends keyof E>(t: K, value: E[K]): boolean
}
