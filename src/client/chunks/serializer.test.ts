import { Serializer, Deserializer } from './serializer'
import { headerSizeBytes, deserializeHeader } from './header'

describe('chunks/serializer', () => {

  describe('constructor', () => {
    it('throws when max message size is <= headerSize', () => {
      expect(() => new Serializer(headerSizeBytes))
      .toThrowError(/should be greater/)
      new Serializer(headerSizeBytes + 1)
    })
  })

  describe('serialize and deserialize', () => {
    function createTestData(size: number, chunkSize: number): Uint8Array {
      const testData = new Uint8Array(size)
      for (let i = 0; i < testData.length; i++) {
        testData[i] = i + 1
      }
      return testData
    }

    it('serializes and deserializes in chunks', () => {
      const size = 11
      const chunkSize = 4
      const s = new Serializer(headerSizeBytes + chunkSize)
      const testData = createTestData(size, chunkSize)
      const chunks = s.serialize(testData)
      expect(chunks.length).toBe(Math.ceil(size / chunkSize))
      const d = new Deserializer()
      expect(d.deserialize(chunks[0])).toBe(null)
      expect(d.deserialize(chunks[1])).toBe(null)
      expect(d.deserialize(chunks[2])).toEqual(testData)
    })

    it('deserializes out of order', () => {
      const size = 11
      const chunkSize = 4
      const s = new Serializer(headerSizeBytes + chunkSize)
      const testData = createTestData(size, chunkSize)
      const chunks = s.serialize(testData)
      expect(chunks.length).toBe(Math.ceil(size / chunkSize))
      const d = new Deserializer()
      expect(d.deserialize(chunks[2])).toBe(null)
      expect(d.deserialize(chunks[0])).toBe(null)
      expect(d.deserialize(chunks[1])).toEqual(testData)
    })

    it('rollovers counter', () => {
      const s = new Serializer()
      ;(s as any).counter = 0xFFFFF
      const data = s.serialize(new Uint16Array(1))
      const h = deserializeHeader(new Uint8Array(data[0]))
      expect(h.messageId).toBe(1)
    })

  })

  it('serializes and deserializes data in chunks', () => {
  })

})
