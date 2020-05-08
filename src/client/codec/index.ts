import { SimpleEmitter } from '../emitter'
import { TextDecoder, TextEncoder } from '../textcodec'
import { decodeHeader, encodeHeader, Header, headerSizeBytes } from './header'

const maxMessageId = 2**16

export interface DataContainer {
  senderId: string
  data: Uint8Array
}

export interface ChunkEvent {
  type: 'data'
  messageId: number
  chunkNum: number
  totalChunks: number
  senderId: string
  chunk: Uint8Array
}

export interface DoneEvent {
  type: 'done'
  senderId: string
  messageId: number
}

export interface ErrorEvent {
  type: 'error'
  error: Error
  senderId: string
  messageId: number
}

export interface EncoderEvents {
  error: ErrorEvent
  data: ChunkEvent
  done: DoneEvent
}

type Values<T, K extends keyof T = keyof T> = T[K]

export interface WebWorker<RecvType, SendType> {
  onmessage: ((e: WorkerMessageEvent<RecvType>) => void) | null
  postMessage: (data: SendType, transfer: Transferable[]) => void
}

export interface WorkerPayload {
  messageId: number
  maxMessageSizeBytes: number
  data: Uint8Array
  senderId: string
  senderIdBytes: Uint8Array
}

export interface WorkerMessageEvent<T> {
  data: T
}

export type EncoderWorker = WebWorker<WorkerPayload, Values<EncoderEvents>>
export type EncoderWorkerShim = WebWorker<Values<EncoderEvents>, WorkerPayload>

type EncodeHeader = typeof encodeHeader

/**
 *
 * A chunk follows a header, and consists of senderId and data.
 *
 */

const workerFunc = (
  encodeHeader: EncodeHeader,
  headerSizeBytes: number,
) => (self: EncoderWorker) => {
  self.onmessage = event => {
    const {
      messageId,
      maxMessageSizeBytes,
      data,
      senderId,
      senderIdBytes,
    } = event.data

    const input = new Uint8Array(data)

    const totalSizeBytes = input.length

    const maxChunkSize = maxMessageSizeBytes - headerSizeBytes

    let readOffset = 0

    const senderIdSizeBytes = senderIdBytes.length
    const dataBytesInChunk = maxChunkSize - senderIdSizeBytes

    if (dataBytesInChunk <= 0) {
      self.postMessage({
        messageId,
        senderId,
        type: 'error',
        error: new Error('Not enough space for data.'),
      }, [])
      return
    }

    const totalChunks = Math.ceil(totalSizeBytes / dataBytesInChunk)

    for (let chunkNum = 0; readOffset < totalSizeBytes; chunkNum++) {
      const chunkSizeBytes = Math
      .min(maxChunkSize, senderIdSizeBytes + totalSizeBytes - readOffset)
      const readSize = chunkSizeBytes - senderIdSizeBytes

      const header: Header = {
        messageId,
        chunkNum,
        totalChunks,
        senderIdSizeBytes,
        chunkSizeBytes,
        totalSizeBytes,
      }

      const headerBytes = encodeHeader(header)
      const chunk = new Uint8Array(headerBytes.length + chunkSizeBytes)

      let writeOffset = 0
      chunk.set(headerBytes, 0)

      writeOffset += headerBytes.length
      chunk.set(senderIdBytes, writeOffset)

      writeOffset += senderIdSizeBytes
      chunk.set(input.slice(readOffset, readOffset + readSize), writeOffset)

      const chunkEvent: ChunkEvent = {
        type: 'data',
        chunkNum: chunkNum,
        totalChunks,
        messageId,
        senderId,
        chunk,
      }

      self.postMessage(chunkEvent, [chunk.buffer])

      readOffset += readSize
    }

    const doneEvent: DoneEvent = {
      type: 'done',
      messageId,
      senderId,
    }

    self.postMessage(doneEvent, [])
  }
}

export class WorkerShim implements EncoderWorkerShim {
  protected readonly instance: EncoderWorker
  onmessage: EncoderWorkerShim['onmessage'] = null

  constructor(readonly initWorker: (self: EncoderWorker) => void) {
    this.instance = {
      onmessage: null,
      postMessage: this.handlePostMessage,
    }
    initWorker(this.instance)
  }

  protected handlePostMessage: EncoderWorker['postMessage'] = data => {
    this.onmessage && this.onmessage({ data })
  }

  postMessage(data: WorkerPayload) {
    this.instance.onmessage && this.instance.onmessage({ data })
  }
}

export class Encoder extends SimpleEmitter<EncoderEvents> {
  protected counter = 0
  protected readonly textEncoder: TextEncoder
  protected readonly worker: EncoderWorkerShim

  protected readonly workerBlobURL?: string

  constructor(readonly maxMessageSizeBytes = 2**16 - 1) {
    super()
    if (maxMessageSizeBytes <= headerSizeBytes) {
      throw new Error('maxMessageSizeBytes should be greater than headerSize')
    }
    this.textEncoder = new TextEncoder()

    try {
      this.workerBlobURL = URL.createObjectURL(
        new Blob(
          ['(', workerFunc.toString(), ')(' +
            encodeHeader.toString() + ', ' + headerSizeBytes+ ')(self)'],
          {type: 'application/javascript'},
        ),
      )
      this.worker = new Worker(this.workerBlobURL) as EncoderWorkerShim
    } catch (err) {
      this.worker = new WorkerShim(workerFunc(encodeHeader, headerSizeBytes))
    }

    this.worker.onmessage = event => {
      const payload = event.data
      this.emit(payload.type, payload)
    }
  }

  async waitFor(messageId: number) {
    // eslint-disable-next-line
    const self = this

    return new Promise<DoneEvent>((resolve, reject) => {
      function handleDone (event: DoneEvent) {
        if (event.messageId === messageId) {
          self.removeListener('done', handleDone)
          self.removeListener('error', handleError)
          resolve(event)
        }
      }

      function handleError (event: ErrorEvent) {
        if (event.messageId === messageId) {
          self.removeListener('done', handleDone)
          self.removeListener('error', handleError)
          reject(event.error)
        }
      }

      self.on('done', handleDone)
      self.on('error', handleError)
    })
  }

  encode(dataContainer: DataContainer): number {

    const { maxMessageSizeBytes } = this
    const { senderId, data } = dataContainer
    const senderIdBytes = this.textEncoder.encode(senderId)

    ++this.counter
    if (this.counter >= maxMessageId) {
      this.counter = 1
    }
    const messageId = this.counter

    const payload: WorkerPayload = {
      messageId,
      maxMessageSizeBytes,
      senderIdBytes,
      senderId,
      data,
    }

    setTimeout(() => {
      this.worker
      .postMessage(payload, [
        data.buffer,
        senderIdBytes.buffer,
      ])
    })

    return messageId
  }
}

interface ChunkContainer {
  senderId: string
  data: Uint8Array
  chunksReceived: number
  totalChunks: number
}

export class Decoder {
  chunksByMessageId: Record<string, ChunkContainer> = {}
  textDecoder: TextDecoder

  constructor() {
    this.textDecoder = new TextDecoder('utf-8')
  }

  decode(data: ArrayBuffer): DataContainer | null {
    const input = new Uint8Array(data)

    const header = decodeHeader(input)

    const dataStart = headerSizeBytes + header.senderIdSizeBytes
    const dataSize = header.chunkSizeBytes - header.senderIdSizeBytes

    const senderId = this.textDecoder.decode(
      input.slice(headerSizeBytes, dataStart))

    const id = senderId + '_' + header.messageId

    const container = this.chunksByMessageId[id] =
      this.chunksByMessageId[id] || {
        senderId,
        data: new Uint8Array(header.totalSizeBytes),
        chunksReceived: 0,
        totalChunks: header.totalChunks,
      }

    const isLastChunk = header.chunkNum === header.totalChunks - 1

    // the last item could have a different chunk size so offset needs to be
    // calculated from the end.
    const offset = isLastChunk
      ? container.data.length - dataSize
      : header.chunkNum * dataSize

    container.data.set(input.slice(dataStart), offset)
    container.chunksReceived += 1

    if (container.chunksReceived === container.totalChunks) {
      // all chunks are received, return the data
      const { data, senderId } = container
      delete this.chunksByMessageId[id]
      return { data, senderId }
    }

    // null signals some chunks are yet to be received
    return null
  }
}
