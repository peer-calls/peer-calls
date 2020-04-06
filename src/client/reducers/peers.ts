import _debug from 'debug'
import forEach from 'lodash/forEach'
import omit from 'lodash/omit'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'
import * as constants from '../constants'
import { MediaStreamAction } from '../actions/MediaActions'
import { RemoveStreamAction, StreamType } from '../actions/StreamActions'
import { HangUpAction } from '../actions/CallActions'

const debug = _debug('peercalls')

export type PeersState = Record<string, Peer.Instance>

const defaultState: PeersState = {}

let localStreams: Record<StreamType, MediaStream | undefined> = {
  camera: undefined,
  desktop: undefined,
}

function removeTrackFromPeer(
  peer: Peer.Instance,
  track: MediaStreamTrack,
  stream: MediaStream,
) {
  try {
    peer.removeTrack(track, stream)
  } catch (err) {
    debug('peer.removeTrack: %s', err)
  }
}

function handleRemoveStream(
  state: PeersState,
  action: RemoveStreamAction,
): PeersState {
  const stream = action.payload.stream
  if (action.payload.userId === constants.ME) {
    forEach(state, peer => {
      stream.getTracks().forEach(track => {
        removeTrackFromPeer(peer, track, stream)
      })
    })
  }

  return state
}

function handleMediaStream(
  state: PeersState,
  action: MediaStreamAction,
): PeersState {
  if (action.status !== 'resolved') {
    return state
  }
  const streamType = action.payload.type
  if (
    action.payload.userId === constants.ME &&
    streamType
  ) {
    forEach(state, peer => {
      const localStream = localStreams[streamType]
      localStream && localStream.getTracks().forEach(track => {
        removeTrackFromPeer(peer, track, localStream)
      })
      const stream = action.payload.stream
      stream.getTracks().forEach(track => {
        peer.addTrack(track, stream)
      })
    })
    localStreams[streamType] = action.payload.stream
  }
  return state
}

export default function peers(
  state = defaultState,
  action: PeerAction | MediaStreamAction | RemoveStreamAction | HangUpAction,
): PeersState {
  switch (action.type) {
    case constants.PEER_ADD:
      return {
        ...state,
        [action.payload.userId]: action.payload.peer,
      }
    case constants.PEER_REMOVE:
      return omit(state, [action.payload.userId])
    case constants.HANG_UP:
      localStreams = {
        camera: undefined,
        desktop: undefined,
      }
      setTimeout(() => {
        forEach(state, peer => peer.destroy())
      })
      return defaultState
    case constants.STREAM_REMOVE:
      return handleRemoveStream(state, action)
    case constants.MEDIA_STREAM:
      return handleMediaStream(state, action)
    default:
      return state
  }
}
