import * as constants from '../constants.js'

export const addMessage = ({ userId, message, timestamp, image }) => ({
  type: constants.MESSAGE_ADD,
  payload: {
    userId,
    message,
    timestamp,
    image
  }
})
