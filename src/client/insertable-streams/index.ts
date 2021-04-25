import _debug from 'debug'
import { PubTrackEvent as PubTrackEvt, TrackKind } from '../SocketEvent'
import { WebWorker } from '../webworker'
import { config } from '../window'


const debug = _debug('peercalls')
const { peerId } = config

interface EncryptStreamEvent {
  type: 'encrypt'
  readable: ReadableStream<RTCEncodedFrame>
  writable: WritableStream<RTCEncodedFrame>
}

interface DecryptStreamEvent {
  type: 'decrypt'
  streamId: string
  kind: TrackKind
  peerId: string
  streams?: RTCInsertableStreams
}

interface PasswordEvent {
  type: 'password'
  password: string
}

interface PubTrackEvent {
  type: 'pubTrack'
  pubTrackEvent: PubTrackEvt
}

interface InitEvent {
  type: 'init'
  peerId: string
  url: string
}

type PostMessageEvent =
  EncryptStreamEvent |
  DecryptStreamEvent |
  PubTrackEvent |
  PasswordEvent |
  InitEvent

export type EncryptionWorker = WebWorker<PostMessageEvent, void>

export interface DecryptParams {
  receiver: RTCRtpReceiver
  kind: TrackKind
  streamId: string
  peerId: string
}

interface WorkerParams {
  peerId: string
  url: string
}

const workerFunc = () => (self: EncryptionWorker) => {
  const params: WorkerParams = {
    url: '',
    peerId: '',
  }

  const tagLength = 128
  const ivByteLength = 16

  interface DecryptContext {
    key?: CryptoKey
    streamProps: StreamProps
    peerId: string
    password: string
  }

  interface EncryptContext {
    key?: CryptoKey
    password: string
  }

  interface Context {
    password: string
    decryptContextByStreamKey: Record<StreamKey, DecryptContext>
    encryptContext: EncryptContext
  }

  const frameTypeToCryptoOffset = {
    key: 10,
    delta: 3,
    empty: 1,
    undefined: 1,
  }

  type StreamKey = string

  interface StreamProps {
    streamId: string
    kind: TrackKind
  }

  function newStreamKey(params: StreamProps): StreamKey {
    const { streamId, kind } = params
    return (streamId + ':' + kind) as StreamKey
  }

  const context: Context = {
    password: '',
    decryptContextByStreamKey: {},
    encryptContext: {
      password: '',
    },
  }

  function createKey(password: string, url: string, peerId: string) {
    const urlBytes = new TextEncoder().encode(url)
    const peerIdBytes = new TextEncoder().encode(peerId)

    const salt = new Uint8Array(
      urlBytes.byteLength + 1 + peerIdBytes.byteLength,
    )
    salt.set(urlBytes)
    salt.set(peerIdBytes, urlBytes.byteLength + 1)

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

  function updateDecryptContext(
    streamProps: StreamProps,
    peerId: string,
  ) {
    const { password } = context
    const { url } = params

    const streamKey = newStreamKey(streamProps)

    let c = context.decryptContextByStreamKey[streamKey]

    // Check if everything is already configured as it should be
    if (c && c.peerId === peerId && c.password === password) {
      return
    }

    c = context.decryptContextByStreamKey[streamKey] = {
      peerId,
      password,
      streamProps,
    }

    if (!password) {
      c.key = undefined
      return
    }

    return createKey(password, url, peerId)
    .then(key => {
      // Ensure there was no context update in the meantime. This can happen
      // if metadata has changed, or a user has left and another one who just
      // joined got assigned the same key.
      if (context.decryptContextByStreamKey[streamKey] === c) {
        context.decryptContextByStreamKey[streamKey].key = key
      }
    })
  }

  function updateEncryptContext() {
    const { password } = context
    const { url, peerId } = params

    if (context.encryptContext &&
        context.encryptContext.password === password) {
      return Promise.resolve()
    }

    if (!password) {
      context.encryptContext.key = undefined
      return Promise.resolve()
    }

    context.encryptContext.password = password

    return createKey(password, url, peerId)
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

    return !!frame && !!(frame as {type: RTCEncodedVideoFrameType}).type
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
    streamProps: StreamProps,
    frame: T,
    controller: TransformStreamDefaultController<T>,
  ) {
    const streamKey = newStreamKey(streamProps)
    const ctx = context.decryptContextByStreamKey[streamKey]
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

  function handlePubTrackEvent(msg: PubTrackEvent) {
    const { pubTrackEvent } = msg
    const { kind, peerId, trackId: { streamId } } = pubTrackEvent
    const streamProps = { kind, streamId }
    const streamKey = newStreamKey(streamProps)

    // We use peerId becaues the peerId is the source of the track and the only
    // one who might use the insertable streams.

    switch (pubTrackEvent.type) {
    case 1: // Pub
      return updateDecryptContext(streamProps, peerId)
    case 2: // Unpub
      delete context.decryptContextByStreamKey[streamKey]
      break
    }
  }

  function handlePassword(msg: PasswordEvent) {
    context.password = msg.password
    // Regenerate all keys
    updateEncryptContext()
    .then(() => Promise.all(
      Object.keys(context.decryptContextByStreamKey).map(streamKey => {
        const decryptContext = context.decryptContextByStreamKey[streamKey]
        const { streamProps } = decryptContext
        return updateDecryptContext(streamProps, decryptContext.peerId)
      }),
    ))
  }

  function handleEncrypt(msg: EncryptStreamEvent) {
    msg.readable
    .pipeThrough(new TransformStream({
      transform: (frame, ctrl) => encrypt(frame, ctrl),
    }))
    .pipeTo(msg.writable)

    return updateEncryptContext()
  }

  function handleDecrypt(msg: DecryptStreamEvent) {
    if (msg.streams) {
      msg.streams.readable
      .pipeThrough(new TransformStream({
        transform: (frame, ctrl) =>
          decrypt(msg, frame, ctrl),
      }))
      .pipeTo(msg.streams.writable)
    }

    return updateDecryptContext(msg, msg.peerId)
  }

  self.onmessage = event => {
    const msg = event.data
    switch (msg.type) {
      case 'init':
        params.url = msg.url
        params.peerId = msg.peerId
        console.log('InsertableStreams worker initialized', params.url)
        break
      case 'password':
        return handlePassword(msg)
      case 'pubTrack':
        return handlePubTrackEvent(msg)
      case 'encrypt':
        return handleEncrypt(msg)
      case 'decrypt':
        return handleDecrypt(msg)
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
      peerId,
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

  postPubTrackEvent(pubTrackEvent: PubTrackEvt) {
    const message: PubTrackEvent = {
      type: 'pubTrack',
      pubTrackEvent,
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
      readable: streams.readable,
      writable: streams.writable,
    }

    this.worker.postMessage(message, [
      streams.readable,
      streams.writable,
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
        readable: encodedStreams.readable,
        writable: encodedStreams.writable,
      }

      transferables.push(streams.readable as unknown as Transferable)
      transferables.push(streams.writable as unknown as Transferable)

      this.sendersReceivers.add(params.receiver)
    }

    const message: DecryptStreamEvent = {
      type: 'decrypt',
      streamId: params.streamId,
      kind: params.kind,
      peerId: params.peerId,
      streams,
    }

    this.worker.postMessage(message, transferables)

    return true
  }
}

export const insertableStreamsCodec = new InsertableStreamsCodec()
