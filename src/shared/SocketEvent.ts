import { TypedEmitter, TypedEmitterKeys } from './TypedEmitter'
import { SignalData } from 'simple-peer'

export interface User {
  socketId: string
  userId?: string
}

export interface Ready {
  room: string
  userId: string
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
  ready: Ready
}

export type ServerSocket =
  Omit<SocketIO.Socket, TypedEmitterKeys> &
  TypedEmitter<SocketEvent>

export type TypedIO = SocketIO.Server & {
  to(roomName: string): TypedEmitter<SocketEvent>
}
