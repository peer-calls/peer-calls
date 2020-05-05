import _debug from 'debug'
import forEach from 'lodash/forEach'
import omit from 'lodash/omit'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'
import * as constants from '../constants'
import { MediaStreamAction, MediaTrackAction, MediaTrackEnableAction, MediaKind } from '../actions/MediaActions'
import { RemoveLocalStreamAction, StreamType } from '../actions/StreamActions'
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

function handleRemoveLocalStream(
  state: PeersState,
  action: RemoveLocalStreamAction,
): PeersState {
  const stream = action.payload.stream

  forEach(state, peer => {
    stream.getTracks().forEach(track => {
      removeTrackFromPeer(peer, track, stream)
    })
  })

  return state
}

function handleLocalMediaStream(
  state: PeersState,
  action: MediaStreamAction,
): PeersState {
  if (action.status !== 'resolved') {
    return state
  }
  const streamType = action.payload.type

  forEach(state, peer => {
    const localStream = localStreams[streamType]
    localStream && localStream.getTracks().forEach(track => {
      removeTrackFromPeer(peer, track, localStream)
    })
    const stream = action.payload.stream
    stream.getTracks().forEach(track => {
      debug(
        'Add track to peer, id: %s, kind: %s, label: %s',
        track.id, track.kind, track.label,
      )
      peer.addTrack(track, stream)
    })
  })
  localStreams[streamType] = action.payload.stream

  return state
}

function getTracksByKind(stream: MediaStream, kind: MediaKind) {
  return kind === 'video' ? stream.getVideoTracks() : stream.getAudioTracks()
}

export function handleLocalMediaTrack(
  state: PeersState,
  action: MediaTrackAction,
): PeersState {
  if (action.status !== 'resolved') {
    return state
  }

  const { payload } = action
  const newTrack = payload.track

  // right now this will work only for StreamTypeCamera
  const localStream = localStreams[payload.type]

  if (!localStream) {
    // this should never happen as we will be in a call already and our
    // version of getUserMedia will return an empty media stream if
    // camera permissions weren't granted by the time we joined the call.
    //
    // This won't work for desktop stream, but this method is not for
    // desktop stream.
    return state
  }

  const oldTrack = getTracksByKind(
    localStream, payload.kind,
  )[0] as MediaStreamTrack | undefined

  if (oldTrack) {
    if (newTrack) {
      forEach(state, peer => {
        peer.replaceTrack(oldTrack, newTrack, localStream)
      })
      localStream.removeTrack(oldTrack)
      localStream.addTrack(newTrack)
      oldTrack.stop()
      return state
    }

    // old track and no new track, mute the current track
    oldTrack.enabled = false
    return state
  }

  if (newTrack) {
    forEach(state, peer => {
      peer.addTrack(newTrack, localStream)
    })
    localStream.addTrack(newTrack)
    return state
  }

  // no old track and no new track, do nothing
  return state
}

export function handleLocalMediaTrackEnable(
  state: PeersState,
  action: MediaTrackEnableAction,
): PeersState {
  const { payload } = action
  const localStream = localStreams[payload.type]
  if (!localStream) {
    return state
  }

  const tracks = getTracksByKind(localStream, payload.kind)
  tracks.forEach(t => t.enabled = true)

  return state
}

export default function peers(
  state = defaultState,
  action:
    PeerAction |
    MediaStreamAction |
    MediaTrackAction |
    MediaTrackEnableAction |
    RemoveLocalStreamAction |
    HangUpAction,
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
      return handleRemoveLocalStream(state, action)
    case constants.MEDIA_STREAM:
      return handleLocalMediaStream(state, action)
    case constants.MEDIA_TRACK:
      return handleLocalMediaTrack(state, action)
    case constants.MEDIA_TRACK_ENABLE:
      return handleLocalMediaTrackEnable(state, action)
    default:
      return state
  }
}
