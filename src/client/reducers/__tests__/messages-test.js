import * as ChatActions from '../../actions/ChatActions.js'
import messages from '../messages.js'

describe('reducers/messages', () => {

  describe('addMessage', () => {
    it('add message to chat', () => {
      const payload = {
        userId: 'test',
        message: 'hello',
        timestamp: new Date(),
        image: null
      }
      let state = messages()
      state = messages(state, ChatActions.addMessage(payload))
      expect(state).toEqual([payload])
    })
  })

  describe('messageHistory', () => {
    it('get chat message hisotry', () => {
      let state = messages()
      state = messages(state, ChatActions.loadHistory([]))
      expect(state).toEqual([])
    })
  })

})
