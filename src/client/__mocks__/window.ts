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

export const play = jest.fn()

export const valueOf = jest.fn()

export const callId = 'call1234'

export const userId = 'user1234'

export const nickname = 'nick1234'

export const network = 'mesh'

export const iceServers = []
