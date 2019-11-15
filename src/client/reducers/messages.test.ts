import * as ChatActions from '../actions/ChatActions'
import messages from './messages'

describe('reducers/messages', () => {

  describe('addMessage', () => {
    it('add message to chat', () => {
      const payload: ChatActions.Message = {
        userId: 'test',
        message: 'hello',
        timestamp: new Date().toLocaleString(),
      }
      let state = messages(undefined, {type: 'test'} as any)
      state = messages(state, ChatActions.addMessage(payload))
      expect(state.list).toEqual([payload])
    })
  })

})
