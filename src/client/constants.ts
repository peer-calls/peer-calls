export const MINIMIZE_TOGGLE = 'MINIMIZE_TOGGLE'
export const MAXIMIZE = 'MAXIMIZE'

export const ALERT = 'ALERT'
export const ALERT_DISMISS = 'ALERT_DISMISS'
export const ALERT_CLEAR = 'ALERT_CLEAR'

export type DialState = 'hung-up' | 'dialling' | 'in-call'

export const DIAL = 'DIAL'
export const DIAL_STATE_HUNG_UP: DialState = 'hung-up'
export const DIAL_STATE_DIALLING: DialState = 'dialling'
export const DIAL_STATE_IN_CALL: DialState = 'in-call'

export const HANG_UP = 'HANG_UP'

export const ME = '_me_'
export const PEERCALLS = '[PeerCalls]'

export const DEVICE_DISABLED_ID = 'disabled'
export const DEVICE_DEFAULT_ID = ''

export const NOTIFY = 'NOTIFY'
export const NOTIFY_DISMISS = 'NOTIFY_DISMISS'
export const NOTIFY_CLEAR = 'NOTIFY_CLEAR'

export const MESSAGE_ADD = 'MESSAGE_ADD'
export const MESSAGE_SEND = 'MESSAGE_SEND'

export const MEDIA_ENUMERATE = 'MEDIA_ENUMERATE'
export const MEDIA_STREAM = 'MEDIA_STREAM'
export const MEDIA_TRACK = 'MEDIA_TRACK'
export const MEDIA_TRACK_ENABLE = 'MEDIA_TRACK_ENABLE'
export const MEDIA_SIZE_CONSTRAINT = 'MEDIA_SIZE_CONSTRAINT'
export const MEDIA_DEVICE_ID = 'MEDIA_DEVICE_ID'
export const MEDIA_DEVICE_TOGGLE = 'MEDIA_DEVICE_TOGGLE'
export const MEDIA_PLAY = 'MEDIA_PLAY'

export const NICKNAMES_SET = 'NICKNAMES_SET'
export const NICKNAME_REMOVE = 'NICKNAME_REMOVE'

export const PEER_ADD = 'PEER_ADD'
export const PEER_REMOVE = 'PEER_REMOVE'
export const PEER_REMOVE_ALL = 'PEER_REMOVE_ALL'
export const PEER_CONNECTED = 'PEER_CONNECTED'

// this data channel must have the same name as the one on the server-side,
// when SFU is used.
export const PEER_DATA_CHANNEL_NAME = 'data'

export const PEER_EVENT_ERROR = 'error'
export const PEER_EVENT_CONNECT = 'connect'
export const PEER_EVENT_CLOSE = 'close'
export const PEER_EVENT_SIGNAL = 'signal'
export const PEER_EVENT_TRACK = 'track'
export const PEER_EVENT_DATA = 'data'

export const PUB_TRACK_EVENT = 'PUB_TRACK_EVENT'

export const SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE =
  'SETTINGS_SHOW_MINIMIZED_TOOLBAR_TOGGLE'
export const SETTINGS_GRID_SET =
  'SETTINGS_GRID_SET'
export const SETTINGS_SHOW_ALL_STATS_TOGGLE =
  'SETTINGS_SHOW_ALL_STATS_TOGGLE'

export const SETTINGS_GRID_AUTO = 'SETTINGS_GRID_AUTO'
export const SETTINGS_GRID_LEGACY = 'SETTINGS_GRID_LEGACY'
export const SETTINGS_GRID_ASPECT = 'SETTINGS_GRID_ASPECT'

export const SIDEBAR_HIDE = 'SIDEBAR_HIDE'
export const SIDEBAR_SHOW = 'SIDEBAR_SHOW'
export const SIDEBAR_TOGGLE = 'SIDEBAR_TOGGLE'

export const SOCKET_CONNECTED = 'SOCKET_CONNECTED'
export const SOCKET_DISCONNECTED = 'SOCKET_DISCONNECTED'
export const SOCKET_EVENT_READY = 'ready'
export const SOCKET_EVENT_SIGNAL = 'signal'
export const SOCKET_EVENT_USERS = 'users'
export const SOCKET_EVENT_HANG_UP = 'hangUp'
export const SOCKET_EVENT_PUB_TRACK = 'pubTrack'
export const SOCKET_EVENT_SUB_TRACK = 'subTrack'

export const STREAM_ADD = 'PEER_STREAM_ADD'
export const STREAM_REMOVE = 'PEER_STREAM_REMOVE'
export const STREAM_TRACK_ADD = 'PEER_STREAM_TRACK_ADD'
export const STREAM_TRACK_REMOVE = 'PEER_STREAM_TRACK_REMOVE'
export const STREAM_DIMENSIONS_SET = 'STREAM_DIMENSIONS_SET'

export const RES_IMG_FIREFOX_SHARE = '/res/ff_share.png'
