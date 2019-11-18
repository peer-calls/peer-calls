import { MediaDevice, AudioConstraint, VideoConstraint, MediaAction, MediaEnumerateAction, MediaStreamAction, MediaPlayAction } from '../actions/MediaActions'
import { MEDIA_ENUMERATE, MEDIA_AUDIO_CONSTRAINT_SET, MEDIA_VIDEO_CONSTRAINT_SET, MEDIA_STREAM, MEDIA_PLAY } from '../constants'

export interface MediaState {
  devices: MediaDevice[]
  video: VideoConstraint
  audio: AudioConstraint
  loading: boolean
  error: string
  autoplayError: boolean
}

const defaultState: MediaState = {
  devices: [],
  video: { facingMode: 'user'},
  audio: true,
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
      return {
        ...state,
        autoplayError: true,
      }
    default:
      return state
  }
}

export default function media(
  state = defaultState,
  action: MediaAction,
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
    default:
      return state
  }
}
