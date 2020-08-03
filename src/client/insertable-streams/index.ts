import { WebWorker } from '../webworker'
import _debug from 'debug'

const debug = _debug('peercalls')

export interface StreamEvent {
  type: 'encrypt' | 'decrypt'
  readableStream: ReadableStream<RTCEncodedFrame>
  writableStream: WritableStream<RTCEncodedFrame>
}

export interface EncryptionKeyEvent {
  type: 'encryption-key'
  encryptionKey: string
}

export type PostMessageEvent = StreamEvent | EncryptionKeyEvent

export type EncryptionWorker = WebWorker<PostMessageEvent, void>

const workerFunc = () => (self: EncryptionWorker) => {
  let encryptionKey = ''

  const frameTypeToCryptoOffset = {
    key: 10,
    delta: 3,
    empty: 1,
    undefined: 1,
  }

  function isVideoFrame(frame: unknown): frame is RTCEncodedVideoFrame {
    // cannot use typeof here because TypeScript converts it into a function
    // and this function call fails.

    // eslint-disable-next-line
    return frame && !!(frame as any).type
  }

  function getCryptoOffset<T extends RTCEncodedFrame>(frame: T): number {
    if (isVideoFrame(frame)) {
      return frameTypeToCryptoOffset[frame.type]
    }

    return frameTypeToCryptoOffset.undefined
  }

  function encrypt<T extends RTCEncodedFrame>(
    frame: T,
    controller: TransformStreamDefaultController<T>,
  ) {
    if (encryptionKey) {
      const view = new DataView(frame.data)
      // Any length that is needed can be used for the new buffer.
      const newData = new ArrayBuffer(frame.data.byteLength + 4)
      const newView = new DataView(newData)

      // const cryptoOffset = 0
      const cryptoOffset = getCryptoOffset(frame)

      for (let i = 0; i < cryptoOffset && i < frame.data.byteLength; i++) {
        newView.setInt8(i, view.getInt8(i))
      }

      // This is a bitwise xor of the key with the payload. This is not strong
      // encryption, just a demo.
      for (let i = cryptoOffset; i < frame.data.byteLength; ++i) {
        const keyByte = encryptionKey.charCodeAt(i % encryptionKey.length)
        newView.setInt8(i, view.getInt8(i) ^ keyByte)
      }

      // Append checksum
      newView.setUint32(frame.data.byteLength, 0xDEADBEEF)

      frame.data = newData
    }
    controller.enqueue(frame)
  }

  function decrypt<T extends RTCEncodedFrame>(
    frame: T,
    controller: TransformStreamDefaultController<T>,
  ) {
    const view = new DataView(frame.data)
    const checksum = frame.data.byteLength > 4
      ? view.getUint32(frame.data.byteLength - 4)
      : false

    if (checksum !== 0xDEADBEEF) {
      controller.enqueue(frame)
      return
    }

    if (encryptionKey) {
      const newData = new ArrayBuffer(frame.data.byteLength - 4)
      const newView = new DataView(newData)

      // const cryptoOffset = 0
      const cryptoOffset = getCryptoOffset(frame)

      for (let i = 0; i < cryptoOffset; ++i) {
        newView.setInt8(i, view.getInt8(i))
      }

      for (let i = cryptoOffset; i < frame.data.byteLength - 4; ++i) {
        const keyByte = encryptionKey.charCodeAt(i % encryptionKey.length)
        newView.setInt8(i, view.getInt8(i) ^ keyByte)
      }
      frame.data = newData

      controller.enqueue(frame)
      return
    }

    frame.data = view.buffer.slice(0, view.buffer.byteLength - 4)
    controller.enqueue(frame)
  }

  self.onmessage = event => {
    const message = event.data
    switch (message.type) {
      case 'encryption-key':
        encryptionKey = message.encryptionKey
        break
      case 'encrypt':
        message.readableStream
        .pipeThrough(new TransformStream({
          transform: encrypt,
        }))
        .pipeTo(message.writableStream)
        break
      case 'decrypt':
        message.readableStream
        .pipeThrough(new TransformStream({
          transform: decrypt,
        }))
        .pipeTo(message.writableStream)
    }
  }
}

export class InsertableStreamsCodec {
  protected worker?: Worker
  protected readonly workerBlobURL?: string

  constructor() {
    if (!(
      typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function'
    )) {
      return
    }
    this.workerBlobURL = URL.createObjectURL(
      new Blob(
        ['(', workerFunc.toString(), ')()(self)'],
        {type: 'application/javascript'},
      ),
    )

    if (
      !(
        typeof RTCRtpSender !== 'undefined' &&
        typeof RTCRtpSender.prototype.createEncodedStreams === 'function' &&
        typeof RTCRtpReceiver !== 'undefined' &&
        typeof RTCRtpReceiver.prototype.createEncodedStreams === 'function'
      )
    ) {
      debug('Environment does not support insertable streams')
      return
    }

    try {
      this.worker = new Worker(this.workerBlobURL)
    } catch (err) {
      debug('Error creating insertable streams worker: %s', err)
    }
  }

  setEncryptionKey(encryptionKey: string): boolean{
    const message: EncryptionKeyEvent = {
      type: 'encryption-key',
      encryptionKey: encryptionKey,
    }

    if (this.worker) {
      this.worker.postMessage(message)
      return true
    }

    return false
  }

  getEncodedStreams(
    senderOrReceiver: RTCRtpSender | RTCRtpReceiver,
  ): RTCInsertableStreams | null {
    if (typeof senderOrReceiver.createEncodedStreams !== 'function') {
      return null
    }

    try {
      return senderOrReceiver.createEncodedStreams!()
    } catch (err) {
      debug('Could not get encoded streams: %s', err)
      return null
    }
  }

  encrypt(sender: RTCRtpSender): boolean {
    if (!this.worker) {
      return false
    }

    const streams = this.getEncodedStreams(sender)
    if (!streams) {
      return false
    }

    const message: StreamEvent = {
      type: 'encrypt',
      readableStream: streams.readableStream,
      writableStream: streams.writableStream,
    }

    this.worker.postMessage(message, [
      streams.readableStream,
      streams.writableStream,
    ] as unknown as Transferable[])

    return true
  }

  decrypt(receiver: RTCRtpReceiver): boolean{
    if (!this.worker) {
      return false
    }

    const streams = this.getEncodedStreams(receiver)
    if (!streams) {
      return false
    }

    const message: StreamEvent = {
      type: 'decrypt',
      readableStream: streams.readableStream,
      writableStream: streams.writableStream,
    }

    this.worker.postMessage(message, [
      streams.readableStream,
      streams.writableStream,
    ] as unknown as Transferable[])

    return true
  }
}

export const insertableStreamsCodec = new InsertableStreamsCodec()
