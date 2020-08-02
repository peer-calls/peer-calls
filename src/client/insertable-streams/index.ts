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

  function encrypt<T extends RTCEncodedFrame>(
    frame: T,
    controller: TransformStreamDefaultController<T>,
  ) {
    if (encryptionKey) {
      const view = new DataView(frame.data)
      // Any length that is needed can be used for the new buffer.
      const newData = new ArrayBuffer(frame.data.byteLength + 5)
      const newView = new DataView(newData)

      // const cryptoOffset = useCryptoOffset
      // ? frameTypeToCryptoOffset[encodedFrame.type]
      // : 0
      const cryptoOffset = 0

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

    if (encryptionKey) {
      if (checksum !== 0xDEADBEEF) {
        console.log('Corrupted frame received checksum',  checksum.toString(16))
        // This can happen when the key is set and there is an unencrypted
        // frame in-flight.
        return
      }

      const newData = new ArrayBuffer(frame.data.byteLength - 4)
      const newView = new DataView(newData)

      // const cryptoOffset = useCryptoOffset
      // ? frameTypeToCryptoOffset[encodedFrame.type]
      // : 0
      const cryptoOffset = 0

      for (let i = 0; i < cryptoOffset; ++i) {
        newView.setInt8(i, view.getInt8(i))
      }

      for (let i = cryptoOffset; i < frame.data.byteLength - 5; ++i) {
        const keyByte = encryptionKey.charCodeAt(i % encryptionKey.length)
        newView.setInt8(i, view.getInt8(i) ^ keyByte)
      }
      frame.data = newData
    }
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

  setEncryptionKey(encryptionKey: string) {
    const message: EncryptionKeyEvent = {
      type: 'encryption-key',
      encryptionKey: encryptionKey,
    }

    if (this.worker) {
      this.worker.postMessage(message)
    }
  }

  encrypt(sender: RTCRtpSender): boolean {
    if (!this.worker) {
      return false
    }

    const streams = sender.createEncodedStreams!()

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

    const streams = receiver.createEncodedStreams!()

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
