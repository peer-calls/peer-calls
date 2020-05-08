import _debug from 'debug'
const debug = _debug('peercalls')

export const createObjectURL = (object: unknown) =>
  window.URL.createObjectURL(object)
export const revokeObjectURL = (url: string) => window.URL.revokeObjectURL(url)

export const valueOf = (id: string) => {
  const el = window.document.getElementById(id) as HTMLInputElement
  return el ? el.value : null
}

export const baseUrl = valueOf('baseUrl')!
export const callId = valueOf('callId')!
export const userId = valueOf('userId')!
export const iceServers = JSON.parse(valueOf('iceServers')!)
export const nickname = valueOf('nickname')!
export const network = valueOf('network')!

debug('Using ice servers: %o', iceServers)

export const MediaStream = window.MediaStream
export const MediaStreamTrack = window.MediaStreamTrack
