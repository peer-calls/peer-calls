import _debug from 'debug'

const debug = _debug('peercalls')

export const createObjectURL = (object: unknown) =>
  window.URL.createObjectURL(object)
export const revokeObjectURL = (url: string) => window.URL.revokeObjectURL(url)

export function play () {
  const videos = window.document.querySelectorAll('video')
  Array.prototype.forEach.call(videos, (video, index) => {
    debug('playing video: %s', index)
    try {
      video.play()
    } catch (e) {
      debug('error playing video: %s', e.name)
    }
  })
}

export const valueOf = (id: string) => {
  const el = window.document.getElementById(id) as HTMLInputElement
  return el && el.value
}

export const baseUrl = valueOf('baseUrl')
export const callId = valueOf('callId')
export const iceServers = JSON.parse(valueOf('iceServers')!)

export const MediaStream = window.MediaStream
