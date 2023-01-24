export const createObjectURL = (object: unknown) =>
  window.URL.createObjectURL(object)
export const revokeObjectURL = (url: string) => window.URL.revokeObjectURL(url)

export const valueOf = (id: string) => {
  const el = window.document.getElementById(id) as HTMLInputElement
  return el ? el.value : null
}

export interface ClientConfig {
  baseUrl: string
  nickname: string
  callId: string
  peerId: string
  peerConfig: PeerConfig
  network: 'mesh' | 'sfu'
}

export interface PeerConfig {
  iceServers: RTCIceServer[]
  encodedInsertableStreams: boolean
}

export const config: ClientConfig  = JSON.parse(valueOf('config')!)

interface CanvasElement extends HTMLCanvasElement {
  captureStream(): MediaStream
}

export const MediaStream = window.MediaStream
export const MediaStreamTrack = window.MediaStreamTrack
export const RTCRtpReceiver = window.RTCRtpReceiver

export const AudioContext = window.AudioContext
export const AudioWorkletNode = window.AudioWorkletNode

export const localStorage = window.localStorage

// createBlackVideoTrack is in window so it can be easily mocked, for example
// jest requires canvas, which requires python to be installed, and that's
// just too much for a simple workaround.
//
// Idea from: https://blog.mozilla.org/webrtc/warm-up-with-replacetrack/
export const createBlackVideoTrack = (
  width: number,
  height: number,
) => {
  const canvas = document.createElement('canvas') as CanvasElement

  canvas.width = width
  canvas.height = height

  canvas.getContext('2d')!.fillRect(0, 0, width, height)

  const stream = canvas.captureStream()

  const [track] = stream.getVideoTracks()

  return track
}

