import { Events, TypedEmitter, SimpleEmitter } from '../emitter'
import { EventEmitter } from 'events'
import _debug from 'debug'

const debug = _debug('peercalls')

export { TypedEmitter }

interface Message {
  type: string
  // room string
  payload: unknown
}

export class SocketClient<E extends Events> extends SimpleEmitter<E> {

  protected readonly emitter = new EventEmitter()
  protected ws!: WebSocket
  protected connected = false
  reconnectTimeout = 2000

  pingIntervalTimeout = 5000
  protected pingInterval: NodeJS.Timeout | undefined

  constructor(readonly url: string) {
    super()
    this.connect()
  }

  protected connect() {
    debug('connecting to: %s', this.url)
    const ws = this.ws = new WebSocket(this.url)

    ws.addEventListener('close', this.wsHandleClose)
    ws.addEventListener('open', this.wsHandleOpen)
    ws.addEventListener('message', this.wsHandleMessage)
  }

  protected wsHandleClose = () => {
    if (this.connected) {
      debug('websocket connection closed')
      this.emitter.emit('disconnect')
      this.connected = false
    } else {
      debug('websocket failed to connect')
    }

    if (this.pingInterval) {
      clearInterval(this.pingInterval)
    }

    if (this.reconnectTimeout) {
      setTimeout(() => this.connect(), this.reconnectTimeout)
    }
  }

  protected wsHandleOpen = () => {
    debug('websocket connected')
    this.connected = true
    this.emitter.emit('connect')

    if (this.pingIntervalTimeout) {
      this.pingInterval = setInterval(this.ping, this.pingIntervalTimeout)
    }
  }

  protected ping = () => {
    this.emit('ping', undefined as E[keyof E])
  }

  protected wsHandleMessage = (e: MessageEvent) => {
    const message: Message = JSON.parse(e.data)
    this.emitter.emit(message.type, message.payload)
  }

  emit<K extends keyof E>(name: K, value: E[K]): void {
    const message: Message = {
      type: name as string,
      payload: value,
    }
    this.ws.send(JSON.stringify(message))
  }
}
