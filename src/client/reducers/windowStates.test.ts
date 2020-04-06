import * as StreamActions from '../actions/StreamActions'
import windowStates from './windowStates'

describe('reducers/windowStates', () => {

  describe('minimizeToggle', () => {
    it('sets windowStates to userId_streamId', () => {
      let state = windowStates(undefined, {type: 'test'} as any)
      const payload = { userId: 'user1', streamId: 'stream1' }
      state = windowStates(state, StreamActions.minimizeToggle(payload))
      expect(state).toEqual({
        'user1_stream1': 'minimized',
      })
      const payload2 = { userId: 'user2', streamId: 'stream2' }
      state = windowStates(state, StreamActions.minimizeToggle(payload2))
      expect(state).toEqual({
        'user1_stream1': 'minimized',
        'user2_stream2': 'minimized',
      })
      state = windowStates(state, StreamActions.minimizeToggle(payload2))
      expect(state).toEqual({
        'user1_stream1': 'minimized',
      })
    })
  })

})
