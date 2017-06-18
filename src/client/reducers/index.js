import active from './active.js'
import alerts from './alerts.js'
import notifications from './notifications.js'
import peers from './peers.js'
import streams from './streams.js'
import { combineReducers } from 'redux'

export default combineReducers({
  active,
  alerts,
  notifications,
  peers,
  streams
})
