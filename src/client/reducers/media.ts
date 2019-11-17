import { MediaDevice, AudioConstraint, VideoConstraint, MediaAction, MediaEnumerateAction, MediaStreamAction } from '../actions/MediaActions'
import { MEDIA_ENUMERATE, MEDIA_AUDIO_CONSTRAINT_SET, MEDIA_VIDEO_CONSTRAINT_SET, MEDIA_VISIBLE_SET, MEDIA_STREAM } from '../constants'

export interface MediaState {
  devices: MediaDevice[]
  video: VideoConstraint
  audio: AudioConstraint
  loading: boolean
  error: string
  visible: boolean
}

const defaultState: MediaState = {
  devices: [],
  video: { facingMode: 'user'},
  audio: true,
  loading: false,
  error: '',
  visible: true,
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
        visible: false,
      }
    case 'rejected':
      return {
        ...state,
        error: action.payload.message,
        visible: true,
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
    case MEDIA_VISIBLE_SET:
      return {
        ...state,
        visible: action.payload.visible,
      }
    case MEDIA_STREAM:
      return handleMediaStream(state, action)
    default:
      return state
  }
}
