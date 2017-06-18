import Promise from 'bluebird'
import _debug from 'debug'

const debug = _debug('peercalls')

export function getUserMedia (constraints) {
  if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
    return navigator.mediaDevices.getUserMedia(constraints)
  }

  return new Promise((resolve, reject) => {
    const getMedia = navigator.getUserMedia || navigator.webkitGetUserMedia
    if (!getMedia) reject(new Error('Browser unsupported'))
    getMedia.call(navigator, constraints, resolve, reject)
  })
}

export const createObjectURL = object => window.URL.createObjectURL(object)

export const navigator = window.navigator

export function play () {
  let videos = window.document.querySelectorAll('video')
  Array.prototype.forEach.call(videos, (video, index) => {
    debug('playing video: %s', index)
    try {
      video.play()
    } catch (e) {
      debug('error playing video: %s', e.name)
    }
  })
}

export const valueOf = id => {
  const el = window.document.getElementById(id)
  return el && el.value
}

export const callId = valueOf('callId')
export const iceServers = JSON.parse(valueOf('iceServers'))
