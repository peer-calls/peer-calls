import * as constants from '../../constants.js'
import active from '../active.js'

describe('reducers/active', () => {

  it('sets active to userId', () => {
    const userId = 'test'
    let state = active()
    state = active(state, {
      type: constants.ACTIVE_SET,
      payload: { userId }
    })
    expect(state).toBe(userId)
  })

})
