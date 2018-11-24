import active from './active.js'
import alerts from './alerts.js'
import notifications from './notifications.js'
import messages from './messages.js'
import peers from './peers.js'
import streams from './streams.js'
import { combineReducers } from 'redux'

export default combineReducers({
  active,
  alerts,
  notifications,
  messages,
  peers,
  streams
})
