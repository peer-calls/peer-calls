// Fix eslint complaining about Instance not being used here.
// eslint-disable-next-line
import { Instance } from 'simple-peer'

declare module 'simple-peer' {
  interface Instance {
    // addTrack is an extended function in jeremija/simple-peer fork that
    // returns a sender.
    addTrack(track: MediaStreamTrack, stream: MediaStream): RTCRtpSender
  }
}
