import { MediaDevice, AudioConstraint, VideoConstraint, MediaAction, MediaEnumerateAction } from '../actions/MediaActions'
import { MEDIA_ENUMERATE, MEDIA_AUDIO_CONSTRAINT_SET, MEDIA_VIDEO_CONSTRAINT_SET } from '../constants'

export interface MediaState {
  devices: MediaDevice[]
  video: VideoConstraint
  audio: AudioConstraint
  loading: boolean
  error: string
}

const defaultState: MediaState = {
  devices: [],
  video: { facingMode: 'user'},
  audio: true,
  loading: false,
  error: '',
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
    default:
      return state
  }
}
