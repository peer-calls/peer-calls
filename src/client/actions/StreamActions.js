import * as constants from '../constants.js'

export const addStream = ({ stream, userId }) => ({
  type: constants.STREAM_ADD,
  payload: {
    userId,
    stream
  }
})

export const removeStream = userId => ({
  type: constants.STREAM_REMOVE,
  payload: { userId }
})

export const activateStream = userId => ({
  type: constants.STREAM_ACTIVATE,
  payload: { userId }
})
