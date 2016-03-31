jest.dontMock('../streamStore.js');
jest.dontMock('debug');

const dispatcher = require('../../dispatcher/dispatcher.js');
const streamStore = require('../streamStore.js');

describe('streamStore', () => {

  let handleAction = dispatcher.register.mock.calls[0][0];
  let onChange = jest.genMockFunction();

  beforeEach(() => {
    onChange.mockClear();
    streamStore.addListener(onChange);
  });
  afterEach(() => streamStore.removeListener(onChange));

  describe('add-stream and remove-stream', () => {

    it('should add a stream', () => {
      let stream = {};

      handleAction({ type: 'add-stream', userId: 'user1', stream });

      expect(streamStore.getStream('user1')).toBe(stream);
      expect(onChange.mock.calls.length).toEqual(1);
    });

    it('should add a stream multiple times', () => {
      let stream1 = {};
      let stream2 = {};

      handleAction({ type: 'add-stream', userId: 'user1', stream: stream1 });
      handleAction({ type: 'add-stream', userId: 'user2', stream: stream2 });

      expect(streamStore.getStream('user1')).toBe(stream1);
      expect(streamStore.getStream('user2')).toBe(stream2);
      expect(streamStore.getStreams()).toEqual({
        user1: stream1,
        user2: stream2
      });
      expect(onChange.mock.calls.length).toEqual(2);
    });

    it('should remove a stream', () => {
      let stream = {};

      handleAction({ type: 'add-stream', userId: 'user1', stream });
      handleAction({ type: 'remove-stream', userId: 'user1' });

      expect(streamStore.getStream('user1')).not.toBeDefined();
      expect(onChange.mock.calls.length).toEqual(2);
    });

  });

});
