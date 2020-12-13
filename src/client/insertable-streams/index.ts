import _debug from 'debug'
import { TrackMetadata } from '../SocketEvent'
import { WebWorker } from '../webworker'
import { config } from '../window'

const debug = _debug('peercalls')
const { userId } = config

interface EncryptStreamEvent {
  type: 'encrypt'
  readableStream: ReadableStream<RTCEncodedFrame>
  writableStream: WritableStream<RTCEncodedFrame>
}

interface DecryptStreamEvent {
  type: 'decrypt'
  // In the case of SFU only transceiver.mid is known at the time of receiving
  // a track until TrackMetadata array is received.
  mid: string
  // In the case of direct P2P connections, userId will be set correctly from
  // the start, but when SFU is used, mid will have to be used to resolve the
  // username after the metadata event is received.
  userId: string
  // The streams property will be undefined if this is a transceiver reuse,
  // but a message still needs to be sent to the worker so that it updates
  // the userId/mid asssociation.
  streams?: RTCInsertableStreams
}

interface PasswordEvent {
  type: 'password'
  password: string
}

interface MetadataEvent {
  type: 'metadata'
  metadata: TrackMetadata[]
}

interface InitEvent {
  type: 'init'
  userId: string
  url: string
}

type PostMessageEvent =
  EncryptStreamEvent |
  DecryptStreamEvent |
  MetadataEvent |
  PasswordEvent |
  InitEvent

export type EncryptionWorker = WebWorker<PostMessageEvent, void>

export interface DecryptParams {
  receiver: RTCRtpReceiver
  // Only transceiver.mid is known at the time of receiving a track.
  mid: string
  userId: string
}

interface WorkerParams {
  userId: string
  url: string
}

const workerFunc = () => (self: EncryptionWorker) => {
  const params: WorkerParams = {
    url: '',
    userId: '',
  }

  const tagLength = 128
  const ivByteLength = 16

  interface DecryptContext {
    key?: CryptoKey
    userId: string
    password: string
  }

  interface EncryptContext {
    key?: CryptoKey
    password: string
  }

  interface Context {
    password: string
    decryptContextByMid: Record<string, DecryptContext>
    encryptContext: EncryptContext
  }

  const frameTypeToCryptoOffset = {
    key: 10,
    delta: 3,
    empty: 1,
    undefined: 1,
  }

  let metadataByMid: Record<string, TrackMetadata> = {}

  const context: Context = {
    password: '',
    decryptContextByMid: {},
    encryptContext: {
      password: '',
    },
  }

  function getUserId(mid: string, userId: string) {
    const metadata = metadataByMid[mid]
    if (!metadata) {
      return userId
    }

    return metadata.userId
  }

  function createKey(password: string, url: string, userId: string) {
    const urlBytes = new TextEncoder().encode(url)
    const userIdBytes = new TextEncoder().encode(userId)

    const salt = new Uint8Array(
      urlBytes.byteLength + 1 + userIdBytes.byteLength,
    )
    salt.set(urlBytes)
    salt.set(userIdBytes, urlBytes.byteLength + 1)

    return crypto.subtle.importKey(
      'raw',
      new TextEncoder().encode(password),
      'PBKDF2',
      false,
      ['deriveKey', 'deriveBits'],
    )
    .then(passwordKey => crypto.subtle.deriveKey(
      {
        name: 'PBKDF2',
        salt,
        iterations: 150000,
        hash: 'SHA-1',
      },
      passwordKey,
      {
        name: 'AES-GCM',
        length: 256,
      },
      false,
      [ 'encrypt', 'decrypt' ],
    ))
  }

  function updateDecryptContext(mid: string, userId: string) {
    const { password } = context
    const { url } = params

    userId = getUserId(mid, userId)

    let c = context.decryptContextByMid[mid]

    if (c && c.userId === userId && c.password === password) {
      return
    }

    c = context.decryptContextByMid[mid] = {
      userId,
      password,
    }

    if (!password) {
      c.key = undefined
      return
    }

    return createKey(password, url, userId)
    .then(key => {
      // Ensure there was no context update in the meantime. This can happen
      // if metadata has changed, or a user has left and another one who just
      // joined got assigned the same mid.
      if (context.decryptContextByMid[mid] === c) {
        context.decryptContextByMid[mid].key = key
      }
    })
  }

  function updateEncryptContext() {
    const { password } = context
    const { url, userId } = params

    if (context.encryptContext &&
        context.encryptContext.password === password) {
      return Promise.resolve()
    }

    if (!password) {
      context.encryptContext.key = undefined
      return Promise.resolve()
    }

    context.encryptContext.password = password

    return createKey(password, url, userId)
    .then(key => {
      // Ensure password has not been changed in the meantime.
      if (context.encryptContext.password === password) {
        context.encryptContext.key = key
      }
    })
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
    const { key } = context.encryptContext

    if (!key) {
      // Encryption key has not yet been created.
      if (!context.password) {
        // Only enqueue the unencrypted frame when there is no password set.
        // If a key was set, but the key creation is in process, we should not
        // leaking unencrypted frames.
        controller.enqueue(frame)
      }
      return
    }

    const iv = new Uint8Array(ivByteLength)
    crypto.getRandomValues(iv)

    const cryptoOffset = getCryptoOffset(frame)
    const additionalData = new Uint8Array(frame.data, 0, cryptoOffset)
    const dataToEncrypt = new Uint8Array(frame.data, cryptoOffset)

    return crypto.subtle.encrypt(
      {
        name: 'AES-GCM',
        iv,
        additionalData,
        tagLength,
      },
      key,
      dataToEncrypt,
    )
    .then(encryptedData => {
      const newData = new Uint8Array(
        additionalData.length + encryptedData.byteLength + iv.byteLength,
      )

      let offset = 0

      newData.set(additionalData, offset)
      offset += additionalData.byteLength

      newData.set(new Uint8Array(encryptedData), offset)
      offset += encryptedData.byteLength

      newData.set(iv, offset)
      offset += iv.byteLength

      frame.data = newData.buffer

      controller.enqueue(frame)
    })
  }

  function decrypt<T extends RTCEncodedFrame>(
    mid: string,
    frame: T,
    controller: TransformStreamDefaultController<T>,
  ) {
    const ctx = context.decryptContextByMid[mid]
    if (!ctx) {
      controller.enqueue(frame)
      return
    }

    if (!ctx.key) {
      controller.enqueue(frame)
      return
    }

    const { key } = ctx

    return Promise.resolve().then(() => {
      const cryptoOffset = getCryptoOffset(frame)

      let offset = 0

      const additionalData = new Uint8Array(frame.data, offset, cryptoOffset)
      offset += additionalData.byteLength

      const encryptedData = new Uint8Array(
        frame.data, offset, frame.data.byteLength - offset - ivByteLength)
      offset += encryptedData.byteLength

      const iv = new Uint8Array(frame.data, offset)

      return crypto.subtle.decrypt(
        {
          name: 'AES-GCM',
          iv,
          additionalData,
          tagLength,
        },
        key,
        encryptedData,
      )
      .then(decryptedData => {
        const data = new Uint8Array(
          additionalData.byteLength + decryptedData.byteLength)

        data.set(additionalData)
        data.set(new Uint8Array(decryptedData), additionalData.byteLength)
        frame.data = data.buffer
      })
    })
    .catch(err => {
      // Decryption with invalid key will throw errors.
    })
    .finally(() => {
      // TODO perhaps it would be wiser not to show unencrypted streams when
      // password is set to ensure users are aware that a receiving stream is
      // not encrypted. Or at least show some kind of warning.
      controller.enqueue(frame)
    })
  }

  function handleMetadata(metadata: TrackMetadata[]) {
    metadataByMid = metadata.reduce((obj, m) => {
      obj[m.mid] = m
      return obj
    }, {} as Record<string, TrackMetadata>)

    const promises = Object.keys(context.decryptContextByMid).map(mid => {
      const m = metadataByMid[mid]

      if (!m) {
        // Delete old key for user which is no longer there.
        delete context.decryptContextByMid[mid]
        return
      }

      return updateDecryptContext(mid, m.userId)
    })

    return Promise.all(promises)
  }

  function handlePassword(msg: PasswordEvent) {
    context.password = msg.password
    // Regenerate all keys
    updateEncryptContext()
    .then(() => Promise.all(
      Object.keys(context.decryptContextByMid).map(mid => {
        const decryptContext = context.decryptContextByMid[mid]
        return updateDecryptContext(mid, decryptContext.userId)
      }),
    ))
  }

  function handleEncrypt(msg: EncryptStreamEvent) {
    msg.readableStream
    .pipeThrough(new TransformStream({
      transform: (frame, ctrl) => encrypt(frame, ctrl),
    }))
    .pipeTo(msg.writableStream)

    return updateEncryptContext()
  }

  function handleDecrypt(msg: DecryptStreamEvent) {
    if (msg.streams) {
      msg.streams.readableStream
      .pipeThrough(new TransformStream({
        transform: (frame, ctrl) =>
          decrypt(msg.mid, frame, ctrl),
      }))
      .pipeTo(msg.streams.writableStream)
    }

    return updateDecryptContext(msg.mid, msg.userId)
  }

  self.onmessage = event => {
    const msg = event.data
    switch (msg.type) {
      case 'init':
        params.url = msg.url
        params.userId = msg.userId
        console.log('InsertableStreams worker initialized', params.url)
        break
      case 'password':
        return handlePassword(msg)
        break
      case 'metadata':
        return handleMetadata(msg.metadata)
        break
      case 'encrypt':
        return handleEncrypt(msg)
        break
      case 'decrypt':
        return handleDecrypt(msg)
        break
    }
  }
}

export class InsertableStreamsCodec {
  protected worker?: Worker
  protected readonly workerBlobURL?: string
  protected sendersReceivers = new Set<RTCRtpSender | RTCRtpReceiver>()

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

    const initMsg: InitEvent = {
      type: 'init',
      url: location.href,
      userId,
    }

    try {
      this.worker = new Worker(this.workerBlobURL)
      this.postMessage(initMsg, [])
    } catch (err) {
      debug('Error creating insertable streams worker: %s', err)
    }
  }

  private postMessage(
    message: PostMessageEvent,
    transfers: Transferable[],
  ): boolean {
    if (!this.worker) {
      return false
    }

    this.worker.postMessage(message, transfers)
    return true
  }

  setPassword(password: string): boolean {
    const message: PasswordEvent = {
      type: 'password',
      password: password,
    }

    return this.postMessage(message, [])
  }

  setTrackMetadata(metadata: TrackMetadata[]): boolean {
    const message: MetadataEvent = {
      type: 'metadata',
      metadata,
    }

    return this.postMessage(message, [])
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

    if (this.sendersReceivers.has(sender)) {
      // This sender has already been seen (transceiver reuse).
      return true
    }

    this.sendersReceivers.add(sender)

    const streams = this.getEncodedStreams(sender)
    if (!streams) {
      return false
    }

    const message: EncryptStreamEvent = {
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

  decrypt(params: DecryptParams): boolean {
    if (!this.worker) {
      return false
    }

    let streams: RTCInsertableStreams | undefined
    const transferables: Transferable[] = []

    // Encoded streams can only be requested once so the following prevents
    // an error during transceiver reuse in the case of the same sender being
    // assigned a different track.
    if (!this.sendersReceivers.has(params.receiver)) {

      const encodedStreams = this.getEncodedStreams(params.receiver)
      if (!encodedStreams) {
        return false
      }

      streams = {
        readableStream: encodedStreams.readableStream,
        writableStream: encodedStreams.writableStream,
      }

      transferables.push(streams.readableStream as unknown as Transferable)
      transferables.push(streams.writableStream as unknown as Transferable)

      this.sendersReceivers.add(params.receiver)
    }

    const message: DecryptStreamEvent = {
      type: 'decrypt',
      mid: params.mid,
      userId: params.userId,
      streams,
    }

    this.worker.postMessage(message, transferables)

    return true
  }
}

export const insertableStreamsCodec = new InsertableStreamsCodec()
