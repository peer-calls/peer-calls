import _debug from 'debug'
import forEach from 'lodash/forEach'
import map from 'lodash/map'
import omit from 'lodash/omit'
import { HangUpAction } from '../actions/CallActions'
import { MediaStreamAction, MediaTrackAction, MediaTrackPayload } from '../actions/MediaActions'
import { RemovePeerAction } from '../actions/PeerActions'
import { AddLocalStreamPayload, AddTrackPayload, PubTrackEventAction, RemoveLocalStreamPayload, RemoveTrackPayload, SetStreamDimensionsAction, StreamAction, StreamDimensionsPayload, StreamType, StreamTypeCamera } from '../actions/StreamActions'
import { audioProcessor } from '../audio'
import { HANG_UP, MEDIA_STREAM, MEDIA_TRACK, PEER_REMOVE, PUB_TRACK_EVENT, STREAM_DIMENSIONS_SET, STREAM_REMOVE, STREAM_TRACK_ADD, STREAM_TRACK_REMOVE } from '../constants'
import { Dim } from '../frame'
import { insertableStreamsCodec } from '../insertable-streams'
import { PubTrack, PubTrackEvent, TrackEventType, TrackKind } from '../SocketEvent'
import { config, createObjectURL, MediaStream, revokeObjectURL } from '../window'
import { RecordSet, removeChild, setChild } from './recordSet'

const debug = _debug('peercalls')

export interface StreamsState {
  localStreams: {
    [t in StreamType]?: LocalStream
  }
  // pubStreamsKeysByPeerId contains a set of keys for pubStreams indexed by
  // the peerId.
  pubStreamsKeysByPeerId: RecordSet<string, string, undefined>
  // pubStreams contains PubStreams indexed by streamId.
  pubStreams: Record<string, PubStream>

  // remoteStreamsKeysByClientId contains a set of keys for remoteStreams
  // indexed by the clientId.
  remoteStreamsKeysByClientId: Record<string, Record<string, undefined>>
  // remoteStreams contains StreamWithURL indexed by streamId.
  remoteStreams: Record<string, StreamWithURL>
}

interface PubStream {
  streamId: string
  peerId: string
  pubTracks: {
    [t in TrackKind]?: PubTrack
  }
}

const defaultState: Readonly<StreamsState> = Object.freeze({
  localStreams: {},
  pubStreamsKeysByPeerId: {},
  pubStreams: {},
  remoteStreamsKeysByClientId: {},
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
  dimensions?: Dim
}

export interface LocalStream extends StreamWithURL {
  type: StreamType
  mirror: boolean
}

export interface PubTracks {
  streamId: string
  tracksByKind: Record<TrackKind, PubTrack>
}

function getDimensionsFromStream(stream: MediaStream) {
  const videos = stream.getVideoTracks()

  if (videos.length === 0) {
    return undefined
  }

  return getDimensionsFromTrack(videos[0])
}

function getDimensionsFromTrack(track: MediaStreamTrack) {
  const settings = track.getSettings()
  const { width, height } = settings

  if (!width || !height) {
    return undefined
  }

  return {
    x: width,
    y: height,
  }
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
    dimensions: getDimensionsFromStream(payload.stream),
  }

  const existingStream = state.localStreams[payload.type]
  if (existingStream) {
    stopStream(existingStream)
  }

  stream.getAudioTracks().forEach(track => {
    // We expect just one audio track in stream.
    audioProcessor.replaceTrack(payload.stream.id, track)
  })

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

  audioProcessor.removeTrack(existing.stream.id)

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

  if (track.kind === 'audio') {
    audioProcessor.removeTrack(streamId)
  }

  if (remoteStream.stream.getTracks().length === 0) {
    stopStream(remoteStream)

    const remoteStreams = omit(state.remoteStreams, streamId)

    const remoteStreamsKeysByClientId = removeChild(
      state.remoteStreamsKeysByClientId,
      peerId,
      streamId,
    )

    return {
      ...state,
      remoteStreams,
      remoteStreamsKeysByClientId,
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
  const { streamId, track, peerId, receiver } = payload

  let remoteStream = state.remoteStreams[streamId]

  if (config.network === 'mesh') {
    // For mesh network, we don't need any special PubTrackEvent, so just act
    // as if we received the PubTrackEvent so we can associate the track with
    // the correct peer.
    state = pubTrack(state, {
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

  const pubStream = state.pubStreams[streamId]

  const originalPeerId = pubStream ? pubStream.peerId : peerId
  insertableStreamsCodec.decrypt({
    receiver,
    kind: track.kind as TrackKind,
    streamId,
    peerId: originalPeerId,
  })

  if (track.kind === 'audio') {
    audioProcessor.addTrack(streamId, track)
  }

  const remoteStreamsKeysByClientId = setChild(
    state.remoteStreamsKeysByClientId,
    peerId,
    streamId,
    undefined,
  )

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
      remoteStreamsKeysByClientId,
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
    remoteStreamsKeysByClientId,
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
  const { kind, peerId } = payload
  const { streamId } = payload.trackId

  const pubStream = state.pubStreams[streamId] || {
    streamId,
    peerId,
    pubTracks: {},
  }

  const pubStreamsKeysByPeerId = setChild(
    state.pubStreamsKeysByPeerId,
    peerId,
    streamId,
    undefined,
  )

  return {
    ...state,
    pubStreamsKeysByPeerId,
    pubStreams: {
      ...state.pubStreams,
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
  const { peerId, kind } = payload
  const { streamId } = payload.trackId

  const pubStream = state.pubStreams[streamId]

  if (!pubStream) {
    // We have some kind of invalid state.
    debug('streams pubTrackRemove: stream not found', streamId, kind)
    return state
  }

  const pubTracks = omit(pubStream.pubTracks, kind)

  // Check if this stream has any other tracks left.
  if (Object.keys(pubTracks).length === 0) {
    return {
      ...state,
      pubStreamsKeysByPeerId: removeChild(
        state.pubStreamsKeysByPeerId,
        peerId,
        streamId,
      ),
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
    state.remoteStreamsKeysByClientId[payload.peerId],
    (_, streamId) => streamId,
  )

  streamIds.forEach(streamId => {
    const stream = state.remoteStreams[streamId]
    if (stream) {
      stopStream(stream)
    }
  })

  const remoteStreamsKeysByClientId =
    omit(state.remoteStreamsKeysByClientId, payload.peerId)
  const remoteStreams = omit(state.remoteStreams, streamIds)

  return {
    ...state,
    remoteStreamsKeysByClientId,
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
    existingStream
  ) {
    return {
     ...state,
      localStreams: {
        ...state.localStreams,
        [type]: {
          ...existingStream,
          mirror: type === StreamTypeCamera &&
            !notMirroredRegexp.test(track.label),
          dimensions: getDimensionsFromTrack(track),
        },
      },
    }
  }

  if (track && track.kind === 'audio') {
    audioProcessor.replaceTrack(type, track)
  }

  return state
}

function setStreamDimensions(
  state: StreamsState,
  payload: StreamDimensionsPayload,
): StreamsState {
  const { streamId, dimensions } = payload

  let remoteStream = state.remoteStreams[streamId]

  if (!remoteStream) {
    return state
  }

  remoteStream = {
    ...remoteStream,
    dimensions,
  }

  return {
    ...state,
    remoteStreams: {
      ...state.remoteStreams,
      [streamId]: remoteStream,
    },
  }
}

export default function streams(
  state: StreamsState = defaultState,
  action:
    StreamAction |
    MediaStreamAction |
    MediaTrackAction |
    HangUpAction |
    PubTrackEventAction |
    RemovePeerAction |
    SetStreamDimensionsAction,
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
    case STREAM_DIMENSIONS_SET:
      return setStreamDimensions(state, action.payload)
    default:
      return state
  }
}
