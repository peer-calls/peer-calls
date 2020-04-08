import { SignalData } from 'simple-peer'

export interface Ready {
  room: string
  userId: string
  nickname: string
}

export interface SocketEvent {
  users: {
    initiator: string
    // peers to connect to
    peerIds: string[]
    // mapping of userId / nickname
    nicknames: Record<string, string>
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
