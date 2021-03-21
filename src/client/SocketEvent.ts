import { SignalData } from 'simple-peer'

export interface Ready {
  room: string
  userId: string
  nickname: string
}

export interface TrackMetadata {
  mid: string
  kind: string
  userId: string
  streamId: string
}

export interface MetadataPayload {
  userId: string
  metadata: TrackMetadata[]
}

export enum TrackEventType {
  Add = 1,
  Remove = 2,
  Sub = 3,
  Unsub = 4,
}

export interface SocketEvent {
  users: {
    initiator: string
    // peers to connect to
    peerIds: string[]
    // mapping of userId / nickname
    nicknames: Record<string, string>
  }
  metadata: MetadataPayload
  hangUp: {
    userId: string
  }
  pubTrack: {
    trackId: number
    pubClientId: string
    userId: string
    type: TrackEventType.Add | TrackEventType.Remove
  }
  subTrack: {
    trackId: number
    pubClientId: string
    type: TrackEventType.Sub | TrackEventType.Unsub
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
