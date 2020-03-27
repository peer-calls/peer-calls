import { Callback, Events, TypedEmitter } from '../../shared'
import { EventEmitter } from 'events'
import _debug from 'debug'

const debug = _debug('peercalls')

interface Message {
  type: string
  // room string
  payload: unknown
}

export class SocketClient<E extends Events> implements TypedEmitter<E> {

  protected readonly emitter = new EventEmitter()
  protected ws!: WebSocket
  reconnectTimeout = 0

  constructor(readonly url: string) {
    this.connect()
  }

  protected connect() {
    const ws = new WebSocket(this.url)

    ws.addEventListener('close', () => {
      debug('websocket connection closed')
      this.emitter.emit('disconnect')
      this.emitter.removeAllListeners()

      if (this.reconnectTimeout) {
        setTimeout(() => this.connect(), this.reconnectTimeout)
      }
    })

    ws.addEventListener('open', () => {
      debug('websocket connected')
      this.emitter.emit('connect')
    })

    ws.addEventListener('message', (e: MessageEvent) => {
      const message: Message = JSON.parse(e.data)
      this.emitter.emit(message.type, message.payload)
    })
  }

  removeAllListeners() {
    this.emitter.removeAllListeners()
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
    const message: Message = {
      type: name as string,
      payload: value,
    }
    this.ws.send(JSON.stringify(message))
  }
}
