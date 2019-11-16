import { TypedEmitter, TypedEmitterKeys } from './TypedEmitter'

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
    signal: any
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
