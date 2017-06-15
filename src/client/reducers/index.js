import alerts from './alerts.js'
import notifications from './notifications.js'
import streams from './streams.js'
import { combineReducers } from 'redux'

export default combineReducers({
  alerts,
  notifications,
  streams
})
