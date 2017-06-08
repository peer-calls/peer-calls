jest.unmock('../streamStore.js')

const createObjectUrl = require('../../browser/createObjectURL.js')
const dispatcher = require('../../dispatcher/dispatcher.js')
const streamStore = require('../streamStore.js')

describe('streamStore', () => {
  let handleAction = dispatcher.register.mock.calls[0][0]
  let onChange = jest.genMockFunction()

  beforeEach(() => {
    onChange.mockClear()
    streamStore.addListener(onChange)
  })
  afterEach(() => streamStore.removeListener(onChange))

  describe('add-stream and remove-stream', () => {
    it('should add a stream', () => {
      let stream = {}

      createObjectUrl.mockImplementation(str => {
        if (str === stream) return 'url1'
      })

      handleAction({ type: 'add-stream', userId: 'user1', stream })

      expect(streamStore.getStream('user1')).toEqual({
        stream,
        url: 'url1'
      })
      expect(onChange.mock.calls.length).toEqual(1)
    })

    it('should add a stream multiple times', () => {
      let stream1 = {}
      let stream2 = {}

      createObjectUrl.mockImplementation(stream => {
        if (stream === stream1) return 'url1'
        if (stream === stream2) return 'url2'
      })

      handleAction({ type: 'add-stream', userId: 'user1', stream: stream1 })
      handleAction({ type: 'add-stream', userId: 'user2', stream: stream2 })

      expect(streamStore.getStream('user1')).toEqual({
        stream: stream1,
        url: 'url1'
      })
      expect(streamStore.getStream('user2')).toEqual({
        stream: stream2,
        url: 'url2'
      })
      expect(streamStore.getStreams()).toEqual({
        user1: {
          stream: stream1,
          url: 'url1'
        },
        user2: {
          stream: stream2,
          url: 'url2'
        }
      })
      expect(onChange.mock.calls.length).toEqual(2)
    })

    it('should remove a stream', () => {
      let stream = {}

      handleAction({ type: 'add-stream', userId: 'user1', stream })
      handleAction({ type: 'remove-stream', userId: 'user1' })

      expect(streamStore.getStream('user1')).not.toBeDefined()
      expect(onChange.mock.calls.length).toEqual(2)
    })
  })
})
