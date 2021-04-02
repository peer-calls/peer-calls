jest.mock('../window')

import { addMessage, MessageType } from '../actions/ChatActions'
import messages, { Message } from './messages'

describe('reducers/messages', () => {

  describe('addMessage', () => {
    it('add message to chat', () => {
      const payload: MessageType = {
        type: 'text',
        peerId: 'test',
        payload: 'hello',
        timestamp: new Date().toISOString(),
      }
      const expected: Message = {
        message: 'hello',
        peerId: 'test',
        timestamp: new Date(payload.timestamp).toLocaleString(),
      }
      let state = messages(undefined, {type: 'test'} as any)
      state = messages(state, addMessage(payload))
      expect(state.list).toEqual([ expected ])
    })
  })

})
