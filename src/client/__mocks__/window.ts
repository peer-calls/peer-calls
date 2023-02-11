import { ClientConfig } from '../window'

export const createObjectURL = jest.fn()
.mockImplementation(object => 'blob://' + String(object))
export const revokeObjectURL = jest.fn()

let count = 0

export class MediaStream {
  readonly id: string
  private tracks: MediaStreamTrack[] = []

  readonly addTrack: (track: MediaStreamTrack) => void
  readonly removeTrack: (track: MediaStreamTrack) => void
  readonly getTracks: () => MediaStreamTrack[]
  readonly getAudioTracks: () => MediaStreamTrack[]
  readonly getVideoTracks: () => MediaStreamTrack[]

  constructor() {
    this.id = String(++count)

    this.addTrack = jest.fn().mockImplementation((track: MediaStreamTrack) => {
      this.tracks.push(track)
    })

    this.removeTrack = jest.fn()
    .mockImplementation((track: MediaStreamTrack) => {
      this.tracks = this.tracks.filter(t => t !== track)
    })

    this.getTracks = jest.fn().mockImplementation(() => {
      return [...this.tracks]
    })

    this.getVideoTracks = jest.fn().mockImplementation(() => {
      return this.tracks.filter(t => t.kind === 'video')
    })

    this.getAudioTracks = jest.fn().mockImplementation(() => {
      return this.tracks.filter(t => t.kind === 'audio')
    })
  }
}

export class MediaStreamTrack {
  kind = 'video'
  enabled = true
  muted = false
  readonly id: string
  readonly stop: () => void

  constructor() {
    this.id = String(++count)
    this.stop = jest.fn()
  }

  getSettings(): MediaTrackSettings {
    return {
      width: 0,
      height: 0,
    }
  }
}

export const navigator = window.navigator

;(window as any).navigator.mediaDevices = {}
window.navigator.mediaDevices.enumerateDevices = async () => {
  return []
}
window.navigator.mediaDevices.getUserMedia = async () => {
  return new MediaStream() as any
}
(window.navigator.mediaDevices as any).getDisplayMedia = async () => {
  return new MediaStream() as any
}

export class RTCRtpReceiver {}

// export const play = jest.fn()

export const valueOf = jest.fn()

export const config: ClientConfig = {
  baseUrl: '',
  callId: 'call1234',
  peerId: 'user1234',
  peerConfig: {
    iceServers: [],
    encodedInsertableStreams: true,
  },
  network: 'sfu',
  nickname: 'nick1234',
}

export class AudioContext {
  audioWorklet: AudioWorklet

  createMediaStreamSource: (
    stream: MediaStream) => MediaStreamTrackAudioSourceNode

  constructor() {
    this.audioWorklet = {
      addModule: jest.fn(),
    }

    this.createMediaStreamSource = jest.fn().mockImplementation(() => {
      return {
        connect: jest.fn(),
        disconnect: jest.fn(),
      }
    })
  }
}

export class AudioWorkletNode {
  port = {}

  constructor(readonly context: BaseAudioContext, readonly name: string) {}
}

export const blackTrack = new MediaStreamTrack()

export const createBlackVideoTrack = (width: number, height: number) => {
  return blackTrack
}
