import { SocketEvent } from './SocketEvent'
import { config } from './window'
import { SocketClient, TypedEmitter } from './ws'
export type ClientSocket = TypedEmitter<SocketEvent>

const wsUrl = location.origin.replace(/^http/, 'ws') +
  config.baseUrl + '/ws/' + config.callId + '/' + config.peerId

export default new SocketClient<SocketEvent>(wsUrl)
