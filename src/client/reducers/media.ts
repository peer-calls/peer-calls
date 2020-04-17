import { ConnectedAction, DialAction, DisconnectedAction, HangUpAction } from '../actions/CallActions'
import { AudioConstraint, MediaAction, MediaDevice, MediaEnumerateAction, MediaPlayAction, MediaStreamAction, VideoConstraint } from '../actions/MediaActions'
import { DIAL, DialState, DIAL_STATE_DIALLING, DIAL_STATE_HUNG_UP, DIAL_STATE_IN_CALL, HANG_UP, MEDIA_AUDIO_CONSTRAINT_SET, MEDIA_ENUMERATE, MEDIA_PLAY, MEDIA_STREAM, MEDIA_VIDEO_CONSTRAINT_SET, SOCKET_CONNECTED, SOCKET_DISCONNECTED } from '../constants'

export interface MediaState {
  socketConnected: boolean
  devices: MediaDevice[]
  video: VideoConstraint
  audio: AudioConstraint
  dialState: DialState
  loading: boolean
  error: string
  autoplayError: boolean
}

const defaultState: MediaState = {
  socketConnected: false,
  devices: [],
  video: { facingMode: 'user'},
  audio: true,
  dialState: DIAL_STATE_HUNG_UP,
  loading: false,
  error: '',
  autoplayError: false,
}

export function handleEnumerate(
  state: MediaState,
  action: MediaEnumerateAction,
): MediaState {
  switch (action.status) {
    case 'resolved':
      return {
        ...state,
        loading: false,
        devices: action.payload,
      }
      case 'pending':
        return {
          ...state,
          loading: true,
        }
      case 'rejected':
        return {
          ...state,
          loading: false,
          error: 'Could not retrieve media devices',
        }
  }
}

export function handleMediaStream(
  state: MediaState,
  action: MediaStreamAction,
): MediaState {
  switch (action.status) {
    case 'resolved':
      return {
        ...state,
      }
    case 'rejected':
      return {
        ...state,
        error: action.payload.message,
      }
    default:
      return state
  }
}

export function handlePlay(
  state: MediaState,
  action: MediaPlayAction,
): MediaState {
  switch (action.status) {
    case 'pending':
    case 'resolved':
      return {
        ...state,
        autoplayError: false,
      }
    case 'rejected':
      console.log('play rejected', action.payload.name)
      if (action.payload.name !== 'NotAllowedError') {
        return state
      }
      return {
        ...state,
        autoplayError: true,
      }
    default:
      return state
  }
}

export function handleDial(state: MediaState, action: DialAction): MediaState {
  switch(action.status) {
    case 'pending':
      return {
        ...state,
        dialState: DIAL_STATE_DIALLING,
      }
    case 'resolved':
      return {
        ...state,
        dialState: DIAL_STATE_IN_CALL,
      }
    case 'rejected':
      return {
        ...state,
        dialState: DIAL_STATE_HUNG_UP,
      }
    default:
      return state
  }
}

export default function media(
  state = defaultState,
  action:
    MediaAction |
    DialAction |
    HangUpAction |
    ConnectedAction |
    DisconnectedAction,
): MediaState {
  switch (action.type) {
    case MEDIA_ENUMERATE:
      return handleEnumerate(state, action)
    case MEDIA_AUDIO_CONSTRAINT_SET:
      return {
        ...state,
        audio: action.payload,
      }
    case MEDIA_VIDEO_CONSTRAINT_SET:
      return {
        ...state,
        video: action.payload,
      }
    case MEDIA_STREAM:
      return handleMediaStream(state, action)
    case MEDIA_PLAY:
      return handlePlay(state, action)
    case DIAL:
      return handleDial(state, action)
    case SOCKET_CONNECTED:
      return {
        ...state,
        socketConnected: true,
      }
    case SOCKET_DISCONNECTED:
      return {
        ...state,
        socketConnected: false,
      }
    case HANG_UP:
      return {
        ...state,
        dialState: DIAL_STATE_HUNG_UP,
      }
    default:
      return state
  }
}
