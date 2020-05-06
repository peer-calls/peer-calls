
/* The header of every chunk is defined below.
 *
 * 0                   1                   2                   3
 * 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |         message id            |         chunk number          |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |0|                         chunk size                          |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |0|                         total size                          |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 * |          total chunks         |
 * +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 *
 * Sizes do not include header size.
 *
 */

export const headerSizeBytes = 14

export interface Header {
  readonly messageId: number
  readonly chunkNum: number
  readonly chunkSizeBytes: number
  readonly totalSizeBytes: number
  readonly totalChunks: number
}

export function serializeHeader(header: Header): Uint8Array {
  const data = new Uint8Array(14)

  data[0] = (header.messageId & 0xFF00) >> 8
  data[1] = header.messageId & 0x00FF

  data[2] = (header.chunkNum & 0xFF00) >> 8
  data[3] = header.chunkNum & 0x00FF

  data[4] = (header.chunkSizeBytes & 0x7F000000) >> 24
  data[5] = (header.chunkSizeBytes & 0x00FF0000) >> 16
  data[6] = (header.chunkSizeBytes & 0x0000FF00) >> 8
  data[7] = header.chunkSizeBytes & 0x000000FF

  data[8] = (header.totalSizeBytes & 0x7F000000) >> 24
  data[9] = (header.totalSizeBytes & 0x00FF0000) >> 16
  data[10] = (header.totalSizeBytes & 0x0000FF00) >> 8
  data[11] = header.totalSizeBytes & 0x000000FF

  data[12] = (header.totalChunks & 0xFF00) >> 8
  data[13] = header.totalChunks & 0x00FF

  return data
}

export function deserializeHeader(data: Uint8Array): Header {
  const messageId = data[0] << 8 | data[1]

  const chunkNum = data[2] << 8 | data[3]

  const chunkSizeBytes =
    data[4] << 24 | data[5] << 16 | data[6] << 8 | data [7]

  const totalSizeBytes =
    data[8] << 24 | data[9] << 16 | data[10] << 8 | data [11]

  const totalChunks = data[12] << 8 | data[13]

  return { messageId, chunkNum, chunkSizeBytes, totalSizeBytes, totalChunks }
}
