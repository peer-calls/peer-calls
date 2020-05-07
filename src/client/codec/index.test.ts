import { Encoder, Decoder } from './index'
import { headerSizeBytes, decodeHeader } from './header'
import { TextDecoder, TextEncoder } from 'util'

describe('chunks/encoder', () => {

  beforeAll(() => {
    window.TextDecoder = TextDecoder as any
    window.TextEncoder = TextEncoder
  })

  describe('constructor', () => {
    it('throws when max message size is <= headerSize', () => {
      expect(() => new Encoder(headerSizeBytes))
      .toThrowError(/should be greater/)
      new Encoder(headerSizeBytes + 1)
    })
  })

  describe('encode and decode', () => {
    function createTestData(size: number): Uint8Array {
      const testData = new Uint8Array(size)
      for (let i = 0; i < testData.length; i++) {
        testData[i] = i + 1
      }
      return testData
    }

    const senderId = 'sender-a'

    it('throws an error when not enough room for data', () => {
      const e = new Encoder(headerSizeBytes + senderId.length)
      const testData = createTestData(1)
      expect(() => e.encode({ senderId, data: testData }))
      .toThrowError(/not enough space/i)
    })

    it('encodes and decodes in chunks', () => {
      const size = 11
      const dataSize = 4
      const chunkSize = senderId.length + dataSize
      const s = new Encoder(headerSizeBytes + chunkSize)
      const testData = createTestData(size)
      const dataContainer = { senderId, data: testData }
      const chunks = s.encode(dataContainer)
      expect(chunks.length).toBe(Math.ceil(size / dataSize))
      const d = new Decoder()
      expect(d.decode(chunks[0])).toBe(null)
      expect(d.decode(chunks[1])).toBe(null)
      expect(d.decode(chunks[2])).toEqual(dataContainer)
    })

    it('decodes out of order', () => {
      const size = 11
      const dataSize = 4
      const chunkSize = senderId.length + dataSize
      const s = new Encoder(headerSizeBytes + chunkSize)
      const testData = createTestData(size)
      const dataContainer = { senderId, data: testData }
      const chunks = s.encode(dataContainer)
      expect(chunks.length).toBe(Math.ceil(size / dataSize))
      const d = new Decoder()
      expect(d.decode(chunks[2])).toBe(null)
      expect(d.decode(chunks[0])).toBe(null)
      expect(d.decode(chunks[1])).toEqual(dataContainer)
    })

    it('rollovers counter', () => {
      const s = new Encoder()
      ;(s as any).counter = 0xFFFFF
      const dataContainer = { senderId, data: new Uint16Array(1) }
      const data = s.encode(dataContainer)
      const h = decodeHeader(new Uint8Array(data[0]))
      expect(h.messageId).toBe(1)
    })

  })

})
