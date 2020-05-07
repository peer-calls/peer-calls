import { EventEmitter } from 'events'
import { Events, TypedEmitter, Callback } from './TypedEmitter'

export abstract class SimpleEmitter<E extends Events>
implements TypedEmitter<E> {
  protected readonly emitter = new EventEmitter()

  removeAllListeners(event?: string) {
    if (arguments.length === 0) {
      this.emitter.removeAllListeners()
    } else {
      this.emitter.removeAllListeners(event)
    }
    // this.ws.removeEventListener('close', this.wsHandleClose)
    // this.ws.removeEventListener('open', this.wsHandleOpen)
    // this.ws.removeEventListener('message', this.wsHandleMessage)
  }

  removeListener<K extends keyof E>(name: K, callback: Callback<E[K]>): this {
    this.emitter.removeListener(name as string, callback)
    return this
  }

  on<K extends keyof E>(name: K, callback: Callback<E[K]>): this {
    this.emitter.on(name as string, callback)
    return this
  }

  once<K extends keyof E>(name: K, callback: Callback<E[K]>): this {
    this.emitter.once(name as string, callback)
    return this
  }

  emit<K extends keyof E>(name: K, value: E[K]): void {
    this.emitter.emit(name as string, value)
  }
}
