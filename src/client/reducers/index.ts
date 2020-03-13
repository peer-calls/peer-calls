import active from './active'
import notifications from './notifications'
import messages from './messages'
import peers from './peers'
import media from './media'
import streams from './streams'
import nicknames from './nicknames'
import { combineReducers } from 'redux'

export default combineReducers({
  active,
  notifications,
  messages,
  media,
  nicknames,
  peers,
  streams,
})
