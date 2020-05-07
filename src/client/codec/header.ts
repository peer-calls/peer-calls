
/* The header of every chunk is defined below.
 *
 * 0                   1                   2                   3
 * 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |         message id            |         chunk number          |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |          total chunks         |         sender id size        |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |0|                chunk size (including sender id)             |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |0|             total data size (excluding sender id)           |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *
 * Sizes do not include header size.
 *
 */

export const headerSizeBytes = 16

export interface Header {
  readonly messageId: number
  readonly chunkNum: number
  readonly totalChunks: number
  readonly senderIdSizeBytes: number
  readonly chunkSizeBytes: number
  readonly totalSizeBytes: number
}

export function encodeHeader(header: Header): Uint8Array {
  const data = new Uint8Array(16)

  data[0] = (header.messageId & 0xFF00) >> 8
  data[1] = header.messageId & 0x00FF

  data[2] = (header.chunkNum & 0xFF00) >> 8
  data[3] = header.chunkNum & 0x00FF

  data[4] = (header.totalChunks & 0xFF00) >> 8
  data[5] = header.totalChunks & 0x00FF

  data[6] = (header.senderIdSizeBytes & 0xFF00) >> 8
  data[7] = header.senderIdSizeBytes & 0x00FF

  data[8] = (header.chunkSizeBytes & 0x7F000000) >> 24
  data[9] = (header.chunkSizeBytes & 0x00FF0000) >> 16
  data[10] = (header.chunkSizeBytes & 0x0000FF00) >> 8
  data[11] = header.chunkSizeBytes & 0x000000FF

  data[12] = (header.totalSizeBytes & 0x7F000000) >> 24
  data[13] = (header.totalSizeBytes & 0x00FF0000) >> 16
  data[14] = (header.totalSizeBytes & 0x0000FF00) >> 8
  data[15] = header.totalSizeBytes & 0x000000FF

  return data
}

export function decodeHeader(data: Uint8Array): Header {
  const messageId = data[0] << 8 | data[1]
  const chunkNum = data[2] << 8 | data[3]
  const totalChunks = data[4] << 8 | data[5]
  const senderIdSizeBytes = data[6] << 8 | data[7]

  const chunkSizeBytes =
    data[8] << 24 | data[9] << 16 | data[10] << 8 | data [11]

  const totalSizeBytes =
    data[12] << 24 | data[13] << 16 | data[14] << 8 | data [15]

  return {
    messageId,
    chunkNum,
    totalChunks,
    senderIdSizeBytes,
    chunkSizeBytes,
    totalSizeBytes,
  }
}
