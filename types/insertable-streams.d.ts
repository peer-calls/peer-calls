/// <reference no-default-lib="true"/>

// These type definitions are taken from
// https://w3c.github.io/webrtc-insertable-streams/

interface RTCConfiguration {
  encodedInsertableStreams?: bool
}

interface RTCRtpSender {
  createEncodedStreams?: () => RTCInsertableStreams
}

interface RTCRtpReceiver {
  createEncodedStreams?: () => RTCInsertableStreams
}

interface RTCInsertableStreams {
  readable: ReadableStream<RTCEncodedFrame>
  writable: WritableStream<RTCEncodedFrame>
}

// New enum for video frame types. Will eventually re-use the equivalent
// defined by WebCodecs.
type RTCEncodedVideoFrameType = 'empty' | 'key' | 'delta'

type RTCEncodedFrame = RTCEncodedVideoFrame | RTCEncodedAudioFrame

interface RTCEncodedVideoFrameMetadata {
  frameId: number
  dependencies: number[]
  width: number
  height: number
  spatialIndex: number
  temporalIndex: number
  synchronizationSource: number
  contributingSources: number[]
}

// New interfaces to define encoded video and audio frames. Will eventually
// re-use or extend the equivalent defined in WebCodecs.
interface RTCEncodedVideoFrame {
  readonly type: RTCEncodedVideoFrameType
  readonly timestamp: number
  data: ArrayBuffer
  getMetadata(): RTCEncodedVideoFrameMetadata
}

interface RTCEncodedAudioFrameMetadata {
    synchronizationSource: number
    contributingSources: number[]
}

interface RTCEncodedAudioFrame {
  readonly timestamp: number
  data: ArrayBuffer
  getMetadata(): RTCEncodedAudioFrameMetadata
}
