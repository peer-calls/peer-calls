import * as constants from '../constants.js'

export const setPosition = ({ position, userId }) => ({
  type: constants.POSITION_SET,
  payload: {
    userId,
    position
  }
})

export const removePosition = userId => ({
  type: constants.POSITION_REMOVE,
  payload: { userId }
})
