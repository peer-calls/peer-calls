import _debug from 'debug'

const debug = _debug('peercalls')

export function getBrowserFeatures() {
  const audioWorklets =
    typeof AudioContext === 'function' &&
    typeof AudioContext.prototype.createMediaStreamSource === 'function' &&
    typeof AudioWorklet === 'function' &&
    typeof AudioWorklet.prototype.addModule === 'function' &&
    typeof AudioWorkletNode === 'function'
  const media =
    'mediaDevices' in navigator &&
    typeof navigator.mediaDevices === 'object' &&
    typeof navigator.mediaDevices.getUserMedia === 'function' &&
    typeof navigator.mediaDevices.enumerateDevices === 'function'
  const mediaStream =
    typeof MediaStream === 'function' && typeof MediaStreamTrack === 'function'
  const buffers =
    typeof TextEncoder === 'function' && typeof TextDecoder === 'function' &&
    typeof ArrayBuffer === 'function' && typeof Uint8Array === 'function'
  const insertableStreams =
    typeof RTCRtpSender === 'function' &&
    typeof RTCRtpSender.prototype.createEncodedStreams === 'function'
  const webrtc =
    typeof RTCPeerConnection === 'function' &&
    typeof RTCPeerConnection.prototype == 'object' &&
    typeof RTCPeerConnection.prototype.createDataChannel === 'function'
  const websockets =
    typeof WebSocket === 'function'
  const webworkers =
    typeof Worker === 'function'

  const features = {
    audioWorklets,
    media,
    mediaStream,
    buffers,
    insertableStreams,
    webrtc,
    websockets,
    webworkers,
  }

  debug('browser features supported: %o', features)

  return features
}
