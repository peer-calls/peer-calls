import * as StreamActions from '../actions/StreamActions'
import active from './active'

describe('reducers/active', () => {

  describe('setActive', () => {
    it('sets active to userId', () => {
      const userId = 'test'
      let state = active(null, {type: 'test'} as any)
      state = active(state, StreamActions.setActive(userId))
      expect(state).toBe(userId)
    })
  })

  describe('toggleActive', () => {
    it('sets active to userId', () => {
      const userId = 'test'
      let state = active(null, {type: 'test'} as any)
      state = active(state, StreamActions.toggleActive(userId))
      expect(state).toBe(userId)
      state = active(state, StreamActions.toggleActive(userId))
      expect(state).toBe(null)
      state = active(state, StreamActions.toggleActive(userId))
      expect(state).toBe(userId)
    })
  })

})
