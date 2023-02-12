import _debug from 'debug'
import forEach from 'lodash/forEach'
import mapValues from 'lodash/mapValues'
import omit from 'lodash/omit'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'
import * as constants from '../constants'
import { MediaStreamAction, MediaTrackAction, MediaTrackEnableAction, getTracksByKind } from '../actions/MediaActions'
import { RemoveLocalStreamAction, StreamType } from '../actions/StreamActions'
import { HangUpAction } from '../actions/CallActions'
import { insertableStreamsCodec } from '../insertable-streams'

const debug = _debug('peercalls')

export interface PeerState {
  instance: Peer.Instance
  // senders are indexed by MediaStreamTrack.id
  senders: Record<string, RTCRtpSender>
}

export type PeersState = Record<string, PeerState>

const defaultState: PeersState = {}

let localStreams: Record<StreamType, MediaStream | undefined> = {
  camera: undefined,
  desktop: undefined,
}

function removeTrackFromPeer(
  peer: PeerState,
  track: MediaStreamTrack,
  stream: MediaStream,
): PeerState {
  try {
    peer.instance.removeTrack(track, stream)
    const senders = omit(peer.senders, track.id)

    return {
      ...peer,
      senders,
    }
  } catch (err) {
    debug('peer.removeTrack: %s', err)

    return peer
  }
}

function addTrackToPeer(
  peer: PeerState,
  track: MediaStreamTrack,
  stream: MediaStream,
): PeerState {
  debug(
    'Add track to peer, id: %s, kind: %s, label: %s',
    track.id, track.kind, track.label,
  )

  const sender = peer.instance.addTrack(track, stream)
  insertableStreamsCodec.encrypt(sender)

  return {
    ...peer,
    senders: {
      ...peer.senders,
      [track.id]: sender,
    },
  }
}


function handleRemoveLocalStream(
  state: PeersState,
  action: RemoveLocalStreamAction,
): PeersState {
  const stream = action.payload.stream

  return mapValues(state, peer => {
    stream.getTracks().forEach(track => {
      peer = removeTrackFromPeer(peer, track, stream)
    })

    return peer
  })
}

function handleLocalMediaStream(
  state: PeersState,
  action: MediaStreamAction,
): PeersState {
  if (action.status !== 'resolved') {
    return state
  }
  const streamType = action.payload.type

  const newState = mapValues(state, peer => {
    const localStream = localStreams[streamType]
    localStream && localStream.getTracks().forEach(track => {
      removeTrackFromPeer(peer, track, localStream)
    })
    const stream = action.payload.stream
    stream.getTracks().forEach(track => {
      peer = addTrackToPeer(peer, track, stream)
    })

    return peer
  })

  localStreams[streamType] = action.payload.stream

  return newState
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
      const newState = mapValues(state, peer => {
        peer.instance.replaceTrack(oldTrack, newTrack, localStream)

        const sender = peer.senders[oldTrack.id]
        const senders = omit(peer.senders, oldTrack.id)

        return {
          ...peer,
          senders: {
            ...senders,
            [newTrack.id]: sender,
          },
        }
      })
      localStream.removeTrack(oldTrack)
      localStream.addTrack(newTrack)
      oldTrack.stop()

      return newState
    }

    // old track and no new track, mute the current track
    oldTrack.enabled = false
    return state
  }

  if (newTrack) {
    const newState = mapValues(state, peer => {
      return addTrackToPeer(peer, newTrack, localStream)
    })
    localStream.addTrack(newTrack)

    return newState
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

export function removeAllPeers(state: PeersState): PeersState {
  forEach(state, peer => peer.instance.destroy())

  return defaultState
}

export function peerConnected(state: PeersState, peerId: string): PeersState {
  let peer = state[peerId]

  forEach(localStreams, stream => {
    if (!stream) {
      return
    }

    // If the local user pressed join call before this peer has joined the
    // call, now is the time to share local media stream with the peer since
    // we no longer automatically send the stream to the peer.
    stream.getTracks().forEach(track => {
      peer = addTrackToPeer(peer, track, stream)
    })
  })

  return {
    ...state,
    [peerId]: peer,
  }
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
        [action.payload.peerId]: {
          instance: action.payload.peer,
          senders: {},
        },
      }
    case constants.PEER_REMOVE:
      return omit(state, [action.payload.peerId])
    case constants.PEER_CONNECTED:
      return peerConnected(state, action.payload.peerId)
    case constants.HANG_UP:
      localStreams = {
        camera: undefined,
        desktop: undefined,
      }

      return removeAllPeers(state)
    case constants.PEER_REMOVE_ALL:
      return removeAllPeers(state)
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
