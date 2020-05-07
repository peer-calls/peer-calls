import { Header, decodeHeader, encodeHeader, headerSizeBytes } from './header'
import { TextEncoder, TextDecoder } from '../textcodec'

const maxMessageId = 2**16

export interface DataContainer {
  senderId: string
  data: ArrayBuffer
}

/**
 *
 * A chunk follows a header, and consists of senderId and data.
 *
 */

export class Encoder {
  protected counter = 0
  protected textEncoder: TextEncoder

  constructor(readonly maxMessageSizeBytes = 2**16) {
    if (maxMessageSizeBytes <= headerSizeBytes) {
      throw new Error('maxMessageSizeBytes should be greater than headerSize')
    }
    this.textEncoder = new TextEncoder()
  }

  encode(dataContainer: DataContainer): ArrayBuffer[] {
    const { senderId, data } = dataContainer
    const input = new Uint8Array(data)

    const totalSizeBytes = input.length

    const maxChunkSize = this.maxMessageSizeBytes - headerSizeBytes

    if (this.counter >= maxMessageId) {
      this.counter = 0
    }
    const messageId = ++this.counter
    let readOffset = 0

    const chunks: Uint8Array[] = []

    const senderIdBytes = this.textEncoder.encode(senderId)
    const senderIdSizeBytes = senderIdBytes.length
    const dataBytesInChunk = maxChunkSize - senderIdSizeBytes

    if (dataBytesInChunk <= 0) {
      throw new Error('Not enough space for data.')
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

      chunks.push(chunk)

      readOffset += readSize
    }

    return chunks
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
