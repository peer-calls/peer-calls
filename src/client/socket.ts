import { baseUrl, callId, userId } from './window'
import { SocketEvent, TypedEmitter } from '../shared'
import { SocketClient } from './ws'
export type ClientSocket = TypedEmitter<SocketEvent>

export default new SocketClient<SocketEvent>(
  baseUrl + '/ws/' + callId + '/' + userId,
)
