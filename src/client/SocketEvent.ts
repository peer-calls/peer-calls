import { SignalData } from 'simple-peer'

export interface Ready {
  room: string
  peerId: string
  nickname: string
}

export interface TrackMetadata {
  mid: string
  kind: string
  peerId: string
  streamId: string
}

export interface MetadataPayload {
  peerId: string
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
    // mapping of peerId / nickname
    nicknames: Record<string, string>
  }
  metadata: MetadataPayload
  hangUp: {
    peerId: string
  }
  pubTrack: {
    trackId: string
    pubClientId: string
    peerId: string
    type: TrackEventType.Add | TrackEventType.Remove
  }
  subTrack: {
    trackId: string
    pubClientId: string
    type: TrackEventType.Sub | TrackEventType.Unsub
  }
  signal: {
    peerId: string
    // eslint-disable-next-line
    signal: SignalData
  }
  connect: undefined
  disconnect: undefined
  ready: Ready
}
