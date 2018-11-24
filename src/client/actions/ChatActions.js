import * as constants from '../constants.js'
import _ from 'underscore'

export function addMessage ({ userId, message, timestamp }) {
  return {
    type: constants.MESSAGE_ADD,
    payload: {
      id: _.uniqueId('chat'),
      userId,
      message,
      timestamp
    }
  }
}

export function loadHistory (messages) {
  return {
    type: constants.MESSAGES_HISTORY,
    messages
  }
}
