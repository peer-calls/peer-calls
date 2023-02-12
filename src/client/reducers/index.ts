import { combineReducers } from 'redux'
import media from './media'
import messages from './messages'
import nicknames from './nicknames'
import notifications from './notifications'
import peers from './peers'
import receivers from './receivers'
import settings from './settings'
import sidebar from './sidebar'
import streams from './streams'
import windowStates from './windowStates'

export default combineReducers({
  notifications,
  messages,
  media,
  nicknames,
  peers,
  receivers,
  settings,
  sidebar,
  streams,
  windowStates,
})
