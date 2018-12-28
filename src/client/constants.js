export const GOOGLE_MAPS_API_KEY = process.env.APP_GOOGLE_MAPS_API_KEY
export const GOOGLE_MAPS_API_URL = 'https://maps.googleapis.com/maps/api/js?'
  + `key=${GOOGLE_MAPS_API_KEY}`
  + '&v=3.exp&libraries=geometry,drawing,places'

export const ACTIVE_SET = 'ACTIVE_SET'
export const ACTIVE_TOGGLE = 'ACTIVE_TOGGLE'

export const ALERT = 'ALERT'
export const ALERT_DISMISS = 'ALERT_DISMISS'
export const ALERT_CLEAR = 'ALERT_CLEAR'

export const INIT = 'INIT'
export const INIT_PENDING = `${INIT}_PENDING`
export const INIT_FULFILLED = `${INIT}_FULFILLED`
export const INIT_REJECTED = `${INIT}_REJECTED`

export const ME = '_me_'

export const NOTIFY = 'NOTIFY'
export const NOTIFY_DISMISS = 'NOTIFY_DISMISS'
export const NOTIFY_CLEAR = 'NOTIFY_CLEAR'

export const MESSAGE_ADD = 'MESSAGE_ADD'
export const MESSAGES_HISTORY = 'MESSAGES_HISTORY'

export const POSITION_SET = 'POSITION_SET'
export const POSITION_REMOVE = 'POSITION_REMOVE'

export const PEER_ADD = 'PEER_ADD'
export const PEER_REMOVE = 'PEER_REMOVE'
export const PEERS_DESTROY = 'PEERS_DESTROY'

export const PEER_EVENT_ERROR = 'error'
export const PEER_EVENT_CONNECT = 'connect'
export const PEER_EVENT_CLOSE = 'close'
export const PEER_EVENT_SIGNAL = 'signal'
export const PEER_EVENT_STREAM = 'stream'
export const PEER_EVENT_DATA = 'data'

export const SOCKET_EVENT_SIGNAL = 'signal'
export const SOCKET_EVENT_USERS = 'users'
export const SOCKET_EVENT_MESSAGES = 'messages'
export const SOCKET_EVENT_NEW_MESSAGE = 'new_message'
export const SOCKET_EVENT_POSITION = 'position'

export const STREAM_ADD = 'PEER_STREAM_ADD'
export const STREAM_REMOVE = 'PEER_STREAM_REMOVE'
