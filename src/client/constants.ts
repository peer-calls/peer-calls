export const MINIMIZE_TOGGLE = 'MINIMIZE_TOGGLE'

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

export const NOTIFY = 'NOTIFY'
export const NOTIFY_DISMISS = 'NOTIFY_DISMISS'
export const NOTIFY_CLEAR = 'NOTIFY_CLEAR'

export const MESSAGE_ADD = 'MESSAGE_ADD'
export const MESSAGE_SEND = 'MESSAGE_SEND'

export const MEDIA_ENUMERATE = 'MEDIA_ENUMERATE'
export const MEDIA_STREAM = 'MEDIA_STREAM'
export const MEDIA_TRACK = 'MEDIA_TRACK'
export const MEDIA_TRACK_ENABLE = 'MEDIA_TRACK_ENABLE'
export const MEDIA_VIDEO_CONSTRAINT_SET = 'MEDIA_VIDEO_CONSTRAINT_SET'
export const MEDIA_AUDIO_CONSTRAINT_SET = 'MEDIA_AUDIO_CONSTRAINT_SET'
export const MEDIA_PLAY = 'MEDIA_PLAY'

export const NICKNAMES_SET = 'NICKNAMES_SET'
export const NICKNAME_REMOVE = 'NICKNAME_REMOVE'

export const PEER_ADD = 'PEER_ADD'
export const PEER_REMOVE = 'PEER_REMOVE'

// this data channel must have the same name as the one on the server-side,
// when SFU is used.
export const PEER_DATA_CHANNEL_NAME = 'data'

export const PEER_EVENT_ERROR = 'error'
export const PEER_EVENT_CONNECT = 'connect'
export const PEER_EVENT_CLOSE = 'close'
export const PEER_EVENT_SIGNAL = 'signal'
export const PEER_EVENT_TRACK = 'track'
export const PEER_EVENT_DATA = 'data'

export const SOCKET_CONNECTED = 'SOCKET_CONNECTED'
export const SOCKET_DISCONNECTED = 'SOCKET_DISCONNECTED'
export const SOCKET_EVENT_METADATA = 'metadata'
export const SOCKET_EVENT_READY = 'ready'
export const SOCKET_EVENT_SIGNAL = 'signal'
export const SOCKET_EVENT_USERS = 'users'
export const SOCKET_EVENT_HANG_UP = 'hangUp'

export const STREAM_ADD = 'PEER_STREAM_ADD'
export const STREAM_REMOVE = 'PEER_STREAM_REMOVE'
export const STREAM_TRACK_ADD = 'PEER_STREAM_TRACK_ADD'
export const STREAM_TRACK_REMOVE = 'PEER_STREAM_TRACK_REMOVE'

export const TRACKS_METADATA = 'TRACKS_METADATA'
