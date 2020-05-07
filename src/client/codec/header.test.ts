import { Header, encodeHeader, decodeHeader, headerSizeBytes } from './header'

describe('chunks/header', () => {

  describe('encodeHeader and decodeHeader', () => {
    it('serializes header to bytes', () => {
      const chunkSizeBytes = 0x7FABCDEF
      const totalSizeBytes = 0x7FDACD56
      const senderIdSizeBytes = 0xFAEB

      const h: Header = {
        messageId: 0xEFA1,
        chunkNum: 0xABCD,
        totalChunks: Math.ceil(totalSizeBytes / chunkSizeBytes),
        senderIdSizeBytes,
        chunkSizeBytes,
        totalSizeBytes,
      }
      const data = encodeHeader(h)
      expect(data.byteLength).toBe(headerSizeBytes)
      const h2 = decodeHeader(data)
      expect(h2).toEqual(h)
    })
  })
})
