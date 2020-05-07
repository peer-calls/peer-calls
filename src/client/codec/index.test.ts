import { Encoder, Decoder, DataContainer } from './index'
import { headerSizeBytes, decodeHeader } from './header'

describe('chunks/encoder', () => {

  describe('constructor', () => {
    it('throws when max message size is <= headerSize', () => {
      expect(() => new Encoder(headerSizeBytes))
      .toThrowError(/should be greater/)
      new Encoder(headerSizeBytes + 1)
    })
  })

  async function encode(encoder: Encoder, d: DataContainer) {
    const chunks: ArrayBuffer[] = []
    const messageId = encoder.encode(d)
    encoder.on('data', e => {
      if (e.messageId === messageId) {
        chunks.push(e.chunk)
      }
    })
    await encoder.waitFor(messageId)
    return chunks
  }

  describe('encode and decode', () => {
    function createTestData(size: number): Uint8Array {
      const testData = new Uint8Array(size)
      for (let i = 0; i < testData.length; i++) {
        testData[i] = i + 1
      }
      return testData
    }

    const senderId = 'sender-a'

    it('throws an error when not enough room for data', async () => {
      const e = new Encoder(headerSizeBytes + senderId.length)
      const testData = createTestData(1)
      const messageId = e.encode({ senderId, data: testData })
      let error: Error
      try {
        await e.waitFor(messageId)
      } catch (err) {
        error = err
      }
      expect(error!).toBeTruthy()
      expect(error!.message).toMatch(/not enough space/i)
    })

    describe('waitFor', () => {
      it('waits for specific messageId done event', async () => {
        const size = 11
        const dataSize = 4
        const chunkSize = senderId.length + dataSize
        const encoder = new Encoder(headerSizeBytes + chunkSize)
        const testData1 = createTestData(size)
        const testData2 = createTestData(size)
        const messageId1 = encoder.encode({ senderId, data: testData1 })
        const messageId2 = encoder.encode({ senderId, data: testData2 })
        await Promise.all([
          encoder.waitFor(messageId1),
          encoder.waitFor(messageId2),
        ])
      })
      it('waits for specific messageId error event', async () => {
        const size = 11
        const chunkSize = senderId.length
        const encoder = new Encoder(headerSizeBytes + chunkSize)
        const testData1 = createTestData(size)
        const testData2 = createTestData(size)
        const messageId1 = encoder.encode({ senderId, data: testData1 })
        const messageId2 = encoder.encode({ senderId, data: testData2 })
        let error: Error
        try {
          await Promise.race([
            encoder.waitFor(messageId1),
            encoder.waitFor(messageId2),
          ])
        } catch (err) {
          error = err
        }
        expect(error!).toBeTruthy()
      })
    })

    it('encodes and decodes in chunks', async () => {
      const size = 11
      const dataSize = 4
      const chunkSize = senderId.length + dataSize
      const encoder = new Encoder(headerSizeBytes + chunkSize)
      const testData = createTestData(size)
      const dataContainer = { senderId, data: testData }
      const chunks = await encode(encoder, dataContainer)
      expect(chunks.length).toBe(Math.ceil(size / dataSize))
      const d = new Decoder()
      expect(d.decode(chunks[0])).toBe(null)
      expect(d.decode(chunks[1])).toBe(null)
      expect(d.decode(chunks[2])).toEqual(dataContainer)
    })

    it('decodes out of order', async () => {
      const size = 11
      const dataSize = 4
      const chunkSize = senderId.length + dataSize
      const encoder = new Encoder(headerSizeBytes + chunkSize)
      const testData = createTestData(size)
      const dataContainer = { senderId, data: testData }
      const chunks = await encode(encoder, dataContainer)
      expect(chunks.length).toBe(Math.ceil(size / dataSize))
      const d = new Decoder()
      expect(d.decode(chunks[2])).toBe(null)
      expect(d.decode(chunks[0])).toBe(null)
      expect(d.decode(chunks[1])).toEqual(dataContainer)
    })

    it('rollovers counter', async () => {
      const encoder = new Encoder()
      ;(encoder as any).counter = 0xFFFFF
      const dataContainer = { senderId, data: new Uint8Array(1) }
      const chunks = await encode(encoder, dataContainer)
      const h = decodeHeader(new Uint8Array(chunks[0]))
      expect(h.messageId).toBe(1)
    })

  })

})
