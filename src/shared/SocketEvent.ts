import { TypedEmitter, TypedEmitterKeys } from './TypedEmitter'
import { SignalData } from 'simple-peer'

export interface User {
  id: string
}

export interface SocketEvent {
  users: {
    initiator: string
    users: User[]
  }
  signal: {
    userId: string
    // eslint-disable-next-line
    signal: SignalData
  }
  connect: undefined
  disconnect: undefined
  ready: string
}

export type ServerSocket =
  Omit<SocketIO.Socket, TypedEmitterKeys> &
  TypedEmitter<SocketEvent> &
  { room?: string }

export type TypedIO = SocketIO.Server & {
  to(roomName: string): TypedEmitter<SocketEvent>
}
