import SocketIOClient from 'socket.io-client'
import { baseUrl } from './window'
import { TypedEmitterKeys, SocketEvent, TypedEmitter } from '../shared'
export type ClientSocket = Omit<SocketIOClient.Socket, TypedEmitterKeys> &
  TypedEmitter<SocketEvent>
const socket: ClientSocket = SocketIOClient('', { path: baseUrl + '/ws' })
export default socket
