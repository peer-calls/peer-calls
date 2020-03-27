import { baseUrl, callId, userId } from './window'
import { SocketEvent, TypedEmitter } from '../shared'
import { SocketClient } from './ws'
export type ClientSocket = TypedEmitter<SocketEvent>

const wsUrl = location.origin.replace(/^http/, 'ws') +
  baseUrl + '/ws/' + callId + '/' + userId

export default new SocketClient<SocketEvent>(wsUrl)
