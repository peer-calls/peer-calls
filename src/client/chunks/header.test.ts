import { Header, serializeHeader, deserializeHeader } from './header'

describe('chunks/header', () => {

  describe('serializeHeader', () => {
    it('serializes header to bytes', () => {
      const chunkSizeBytes = 0x7FABCDEF
      const totalSizeBytes = 0x7FDACD56

      const h: Header = {
        chunkNum: 0xABCD,
        chunkSizeBytes,
        totalSizeBytes,
        messageId: 0xEFA1,
        totalChunks: Math.ceil(totalSizeBytes / chunkSizeBytes),
      }
      const data = serializeHeader(h)
      const h2 = deserializeHeader(data)
      expect(h2).toEqual(h)
    })
  })
})
