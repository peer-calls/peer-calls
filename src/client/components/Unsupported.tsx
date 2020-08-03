import React from 'react'
import { MdError } from 'react-icons/md'
import { Message } from './Message'
import { getBrowserFeatures } from '../features'

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
  const features = getBrowserFeatures()

  return features.media &&
    features.mediaStream &&
    features.buffers &&
    features.webrtc &&
    features.websockets &&
    features.webworkers
}
