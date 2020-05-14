import _debug from 'debug'
import React from 'react'
import { MdError } from 'react-icons/md'
import { Message } from './Message'

const debug = _debug('peercalls')

export class Unsupported extends React.PureComponent {
  supported = isBrowserSupported()
  render() {
    return !this.supported && (
      <Message className='message-error'>
        <MdError className='icon' />
        <span>
          <strong>You are using an unsupported browser!</strong>
          <br />
          <br />
          If you
          experience any issues during the call, please try joining using the
          latest version of
          {' '}
          <a href="https://www.google.com/chrome/">Chrome</a>,
          {' '}
          <a href="https://www.microsoft.com/en-us/edge">Edge</a>,
          {' '}
          <a href="https://www.mozilla.org/en-US/firefox/new/">Firefox</a>, or
          {' '}
          <a href="https://support.apple.com/en-us/HT204416">Safari</a>.
          <br />
          On iOS devices, only Safari v12+ is supported.
        </span>
      </Message>
    )
  }
}

export function isBrowserSupported() {
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
  const webrtc =
    typeof RTCPeerConnection === 'function' &&
    typeof RTCPeerConnection.prototype == 'object' &&
    typeof RTCPeerConnection.prototype.createDataChannel === 'function'
  const websockets =
    typeof WebSocket === 'function'
  const webworkers =
    typeof Worker === 'function'

  debug(
    'browser features supported: %o',
    { media, mediaStream, buffers, webrtc, websockets, webworkers },
  )

  return media &&
    mediaStream &&
    buffers &&
    webrtc &&
    websockets &&
    webworkers
}
