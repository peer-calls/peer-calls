import _debug from 'debug'
import forEach from 'lodash/forEach'
import map from 'lodash/map'
import omit from 'lodash/omit'
import { HangUpAction } from '../actions/CallActions'
import { MediaStreamAction, MediaTrackAction, MediaTrackPayload } from '../actions/MediaActions'
import { RemovePeerAction } from '../actions/PeerActions'
import { AddLocalStreamPayload, AddTrackPayload, PubTrackEventAction, RemoveLocalStreamPayload, RemoveTrackPayload, StreamAction, StreamType, StreamTypeCamera } from '../actions/StreamActions'
import { HANG_UP, MEDIA_STREAM, MEDIA_TRACK, PEER_REMOVE, PUB_TRACK_EVENT, STREAM_REMOVE, STREAM_TRACK_ADD, STREAM_TRACK_REMOVE } from '../constants'
import { PubTrack, PubTrackEvent, TrackEventType, TrackKind } from '../SocketEvent'
import { createObjectURL, MediaStream, revokeObjectURL, config } from '../window'

import { insertableStreamsCodec } from '../insertable-streams'

const debug = _debug('peercalls')

export interface StreamsState {
  localStreams: {
    [t in StreamType]?: LocalStream
  }
  // pubStreamsKeysByPeerId contains a set of keys for pubStreams indexed by
  // the broadcasterId.
  pubStreamsKeysByPeerId: Record<string, Record<string, true>>
  // pubStreams contains PubStreams indexed by streamId.
  pubStreams: Record<string, PubStream>

  // remoteStreamsKeysByPeerId contains a set of keys for remoteStreams indexed
  // by the broadcasterId.
  remoteStreamsKeysByPeerId: Record<string, Record<string, true>>
  // remoteStreams contains StreamWithURL indexed by streamId.
  remoteStreams: Record<string, StreamWithURL>
}

interface PubStream {
  peerId: string
  pubTracks: {
    [t in TrackKind]?: PubTrack
  }
}

const defaultState: Readonly<StreamsState> = Object.freeze({
  localStreams: {},
  pubStreamsKeysByPeerId: {},
  pubStreams: {},
  remoteStreamsKeysByPeerId: {},
  remoteStreams: {},
})

function safeCreateObjectURL (stream: MediaStream) {
  try {
    return createObjectURL(stream)
  } catch (err) {
    return undefined
  }
}

export interface StreamWithURL {
  stream: MediaStream
  streamId: string
  url?: string
}

export interface LocalStream extends StreamWithURL {
  type: StreamType
  mirror: boolean
}

export interface PubTracks {
  streamId: string
  tracksByKind: Record<TrackKind, PubTrack>
}

function addLocalStream (
  state: StreamsState, payload: AddLocalStreamPayload,
): StreamsState {
  const { stream } = payload
  debug('streams addLocalStream')

  const streamWithURL: LocalStream = {
    stream: payload.stream,
    streamId: payload.stream.id,
    type: payload.type,
    url: safeCreateObjectURL(stream),
    mirror: payload.type === StreamTypeCamera &&
      !!stream.getVideoTracks().find(t => !notMirroredRegexp.test(t.label)),
  }

  const existingStream = state.localStreams[payload.type]
  if (existingStream) {
    stopStream(existingStream)
  }

  return {
    ...state,
    localStreams: {
      ...state.localStreams,
      [payload.type]: streamWithURL,
    },
  }
}

function removeLocalStream (
  state: StreamsState,
  payload: RemoveLocalStreamPayload,
): StreamsState {
  debug('streams removeLocalStream')
  const { localStreams } = state
  const existing = localStreams[payload.streamType]
  if (!existing) {
    return state
  }

  stopStream(existing)
  return {
    ...state,
    localStreams: omit(localStreams, [payload.streamType]),
  }
}

function removeTrack (
  state: Readonly<StreamsState>,
  payload: RemoveTrackPayload,
): StreamsState {
  const { streamId, peerId, track } = payload
  debug('streams removeTrack', streamId, track.id)

  if (config.network === 'mesh') {
    // For mesh network, we don't need any special PubTrackEvent, so just act
    // as if we received the PubTrackEvent so we can associate the track with
    // the correct peer.
    state = pubTrack(state, {
      broadcasterId: peerId,
      peerId,
      pubClientId: peerId,
      trackId: {
        id: track.id,
        streamId,
      },
      kind: track.kind as TrackKind,
      type: TrackEventType.Remove,
    })
  }

  const remoteStream = state.remoteStreams[streamId]
  if (!remoteStream) {
    debug('streams removeTrack stream not found', streamId)
    return state
  }

  // NOTE: we do not remove event listeners from the track because it is
  // possible that it was just temporarily muted.
  remoteStream.stream.removeTrack(track)

  if (remoteStream.stream.getTracks().length === 0) {
    stopStream(remoteStream)

    const remoteStreams = omit(state.remoteStreams, streamId)

    const streamKeys = omit(state.remoteStreamsKeysByPeerId[peerId], streamId)

    const remoteStreamsKeysByPeerId = Object.keys(streamKeys).length === 0
      ?  omit(state.remoteStreamsKeysByPeerId, peerId)
      : {
        ...state.remoteStreamsKeysByPeerId,
        [peerId]: streamKeys,
      }

    return {
      ...state,
      remoteStreams,
      remoteStreamsKeysByPeerId,
    }
  }

  // No need to update state when the existing stream only modified and not
  // removed. The <video> tag should handle this automatically.
  return state
}

function addTrack (
  state: Readonly<StreamsState>,
  payload: AddTrackPayload,
): StreamsState {
  const { streamId, track, peerId } = payload

  let remoteStream = state.remoteStreams[streamId]

  if (config.network === 'mesh') {
    // For mesh network, we don't need any special PubTrackEvent, so just act
    // as if we received the PubTrackEvent so we can associate the track with
    // the correct peer.
    state = pubTrack(state, {
      broadcasterId: peerId,
      peerId,
      pubClientId: peerId,
      trackId: {
        id: track.id,
        streamId,
      },
      kind: track.kind as TrackKind,
      type: TrackEventType.Add,
    })
  }

  const remoteStreamsKeysByPeerId = {
    ...state.remoteStreamsKeysByPeerId,
    [peerId]: {
      ...state.remoteStreamsKeysByPeerId[peerId],
      [streamId]: true as true,
    },
  }

  if (!remoteStream) {
    const stream = new MediaStream()

    remoteStream = {
      stream,
      streamId,
      url: safeCreateObjectURL(stream),
    }

    remoteStream.stream.addTrack(track)

    return {
      ...state,
      remoteStreamsKeysByPeerId,
      remoteStreams: {
        ...state.remoteStreams,
        [streamId]: remoteStream,
      },
    }
  }

  remoteStream.stream.addTrack(track)

  // No need to update state when the existing stream only modified and not
  // added. The <video> tag should handle this automatically.
  return {
    ...state,
    remoteStreamsKeysByPeerId,
  }
}

function stopStream (s: StreamWithURL) {
  debug('streams stopStream()')
  s.stream.getTracks().forEach(track => stopTrack(track))
  s.url && revokeObjectURL(s.url)
}

function stopTrack (track: MediaStreamTrack) {
  track.stop()
  track.onmute = null
  track.onunmute = null
}

function pubTrackAdd (
  state: Readonly<StreamsState>,
  payload: PubTrackEvent,
): StreamsState {
  const { broadcasterId, kind } = payload
  const { streamId } = payload.trackId
  const { pubStreams } = state

  const streamIds = state.pubStreamsKeysByPeerId[broadcasterId] || {}

  const pubStream = state.pubStreams[streamId] || {
    peerId: broadcasterId,
    pubTracks: {},
  }

  return {
    ...state,
    pubStreamsKeysByPeerId: {
      ...state.pubStreamsKeysByPeerId,
      [broadcasterId]: {
        ...streamIds,
        [streamId]: true,
      },
    },
    pubStreams: {
      ...pubStreams,
      [streamId]: {
        ...pubStream,
        pubTracks: {
          ...pubStream.pubTracks,
          [kind]: payload,
        },
      },
    },
  }
}

function pubTrackRemove (
  state: Readonly<StreamsState>,
  payload: PubTrackEvent,
): StreamsState {
  const { broadcasterId, kind } = payload
  const { streamId } = payload.trackId

  let streamIds = state.pubStreamsKeysByPeerId[broadcasterId] || {}

  const pubStream = state.pubStreams[streamId]

  if (!pubStream) {
    // We have some kind of invalid state.
    debug('streams pubTrackRemove: stream not found', streamId, kind)
    return state
  }

  const pubTracks = omit(pubStream.pubTracks, kind)

  // Check if this stream has any other tracks left.
  if (Object.keys(pubTracks).length === 0) {
    // No more tracks left, remove the streamId.
    streamIds = omit(streamIds, streamId)

    // Check if this peer still has any other streams left, and remove its key
    // if it does not.
    const pubStreamsKeysByPeerId = Object.keys(streamIds).length === 0
      ? omit(state.pubStreamsKeysByPeerId, broadcasterId)
      : {
        ...state.pubStreamsKeysByPeerId,
        [broadcasterId]: streamIds,
      }

    return {
      ...state,
      pubStreamsKeysByPeerId,
      remoteStreams: omit(state.remoteStreams, streamId),
    }
  }

  return {
    ...state,
    pubStreams: {
      ...state.pubStreams,
      [streamId]: {
        ...pubStream,
        pubTracks,
      },
    },
  }
}

function pubTrack (
  state: StreamsState,
  payload: PubTrackEvent,
): StreamsState {
  // Maintain association between track metadata and peerId.
  insertableStreamsCodec.postPubTrackEvent(payload)

  switch (payload.type) {
  case TrackEventType.Add:
    state = pubTrackAdd(state, payload)
    break
  case TrackEventType.Remove:
    state = pubTrackRemove(state, payload)
    break
  }

  return state
}

function removePeer(
  state: Readonly<StreamsState>,
  payload: RemovePeerAction['payload'],
): StreamsState {
  debug('streams removePeer: %o', payload)

  const streamIds = map(
    state.remoteStreamsKeysByPeerId[payload.peerId],
    (_, streamId) => streamId,
  )

  streamIds.forEach(streamId => {
    const stream = state.remoteStreams[streamId]
    stopStream(stream)
  })

  const remoteStreamsKeysByPeerId =
    omit(state.remoteStreamsKeysByPeerId, payload.peerId)
  const remoteStreams = omit(state.remoteStreams, streamIds)

  return {
    ...state,
    remoteStreamsKeysByPeerId,
    remoteStreams,
  }
}

const notMirroredRegexp = /back/i

function setLocalStreamMirror(
  state: StreamsState,
  payload: MediaTrackPayload,
): StreamsState {
  const { track, type } = payload
  const existingStream = state.localStreams[type]

  if (
    track &&
    track.kind === 'video' &&
    type === StreamTypeCamera &&
    existingStream
  ) {
    return {
     ...state,
      localStreams: {
        ...state.localStreams,
        [type]: {
          ...existingStream,
          mirror: !notMirroredRegexp.test(track.label),
        },
      },
    }
  }

  return state
}

export default function streams(
  state: StreamsState = defaultState,
  action:
    StreamAction |
    MediaStreamAction |
    MediaTrackAction |
    HangUpAction |
    PubTrackEventAction |
    RemovePeerAction,
): StreamsState {
  switch (action.type) {
    case STREAM_REMOVE:
      return removeLocalStream(state, action.payload)
    case STREAM_TRACK_ADD:
      return addTrack(state, action.payload)
    case STREAM_TRACK_REMOVE:
      return removeTrack(state, action.payload)
    case PUB_TRACK_EVENT:
      return pubTrack(state, action.payload)
    case PEER_REMOVE:
      return removePeer(state, action.payload)
    case HANG_UP:
      forEach(state.localStreams, ls => stopStream(ls!))
      forEach(state.remoteStreams, rs => stopStream(rs))
      // TODO reset insertableStreamsCodec context?
      return defaultState
    case MEDIA_STREAM:
      if (action.status === 'resolved') {
        return addLocalStream(state, action.payload)
      } else {
        return state
      }
    case MEDIA_TRACK:
      if (action.status === 'resolved') {
        return setLocalStreamMirror(state, action.payload)
      } else {
        return state
      }
    default:
      return state
  }
}
