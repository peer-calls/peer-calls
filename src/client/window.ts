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
  userId: string
  iceServers: RTCIceServer[]
  network: 'mesh' | 'sfu'
}

export const config: ClientConfig  = JSON.parse(valueOf('config')!)

export const MediaStream = window.MediaStream
export const MediaStreamTrack = window.MediaStreamTrack
