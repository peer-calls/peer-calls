import * as StreamActions from '../../actions/StreamActions.js'
import active from '../active.js'

describe('reducers/active', () => {

  describe('setActive', () => {
    it('sets active to userId', () => {
      const userId = 'test'
      let state = active()
      state = active(state, StreamActions.setActive(userId))
      expect(state).toBe(userId)
    })
  })

  describe('toggleActive', () => {
    it('sets active to userId', () => {
      const userId = 'test'
      let state = active()
      state = active(state, StreamActions.toggleActive(userId))
      expect(state).toBe(userId)
      state = active(state, StreamActions.toggleActive(userId))
      expect(state).toBe(null)
      state = active(state, StreamActions.toggleActive(userId))
      expect(state).toBe(userId)
    })
  })

})
