import { Header, deserializeHeader, serializeHeader, headerSizeBytes } from './header'

const maxMessageId = 2**16

export class Serializer {
  protected counter = 0

  constructor(readonly maxMessageSizeBytes = 2**16) {
    if (maxMessageSizeBytes <= headerSizeBytes) {
      throw new Error('maxMessageSizeBytes should be greater than headerSize')
    }
  }

  serialize(data: ArrayBuffer): ArrayBuffer[] {
    const input = new Uint8Array(data)

    const totalSizeBytes = input.length

    const maxChunkSize = this.maxMessageSizeBytes - headerSizeBytes
    const totalChunks = Math.ceil(totalSizeBytes / maxChunkSize)

    if (this.counter >= maxMessageId) {
      this.counter = 0
    }
    const messageId = ++this.counter
    let offset = 0

    const chunks: Uint8Array[] = []

    for (let chunkNum = 0; offset < totalSizeBytes; chunkNum++) {
      const chunkSizeBytes = Math.min(maxChunkSize, totalSizeBytes - offset)

      const header: Header = {
        messageId,
        chunkNum,
        chunkSizeBytes,
        totalSizeBytes,
        totalChunks,
      }

      const headerBytes = serializeHeader(header)
      const chunk = new Uint8Array(headerBytes.length + chunkSizeBytes)
      chunk.set(headerBytes, 0)
      chunk.set(
        input.slice(offset, offset + chunkSizeBytes), headerBytes.length)

      chunks.push(chunk)

      offset += chunkSizeBytes
    }

    return chunks
  }
}

interface ChunkContainer {
  data: Uint8Array
  chunksReceived: number
  totalChunks: number
}

export class Deserializer {
  chunksByMessageId: Record<number, ChunkContainer> = {}

  deserialize(data: ArrayBuffer): ArrayBuffer | null {
    const input = new Uint8Array(data)

    const header = deserializeHeader(input.slice(0, headerSizeBytes))

    const container = this.chunksByMessageId[header.messageId] =
      this.chunksByMessageId[header.messageId] || {
        data: new Uint8Array(header.totalSizeBytes),
        chunksReceived: 0,
        totalChunks: header.totalChunks,
      }

    const isLastChunk = header.chunkNum === header.totalChunks - 1

    // the last item could have a different chunk size so offset needs to be
    // calculated from the end.
    const offset = isLastChunk
      ? container.data.length - header.chunkSizeBytes
      : header.chunkNum * header.chunkSizeBytes

    container.data.set(input.slice(headerSizeBytes), offset)
    container.chunksReceived += 1

    if (container.chunksReceived === container.totalChunks) {
      // all chunks are received, return the data
      data = container.data
      delete this.chunksByMessageId[header.messageId]
      return data
    }

    // null signals some chunks are yet to be received
    return null
  }
}
