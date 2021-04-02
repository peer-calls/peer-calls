import _debug from 'debug'
import forEach from 'lodash/forEach'
import keyBy from 'lodash/keyBy'
import omit from 'lodash/omit'
import { MetadataPayload, TrackMetadata } from '../SocketEvent'
import { HangUpAction } from '../actions/CallActions'
import { MediaTrackAction, MediaStreamAction, MediaTrackPayload } from '../actions/MediaActions'
import { NicknameRemoveAction, NicknameRemovePayload } from '../actions/NicknameActions'
import { RemovePeerAction } from '../actions/PeerActions'
import { AddLocalStreamPayload, AddTrackPayload, RemoveLocalStreamPayload, StreamAction, StreamType, TracksMetadataAction, StreamTypeCamera } from '../actions/StreamActions'
import { HANG_UP, MEDIA_STREAM, NICKNAME_REMOVE, PEER_REMOVE, STREAM_REMOVE, STREAM_TRACK_ADD, STREAM_TRACK_REMOVE, TRACKS_METADATA, MEDIA_TRACK } from '../constants'
import { createObjectURL, MediaStream, revokeObjectURL } from '../window'

const debug = _debug('peercalls')
const defaultState = Object.freeze({
  localStreams: {},
  streamsByPeerId: {},
  metadataByPeerIdMid: {},
  trackIdToPeerIdMid: {},
  tracksByPeerIdMid: {},
})

const peerIdMidSeparator = '::'

function getPeerIdMid(peerId: string, mid: string): string {
  return peerId + peerIdMidSeparator + mid
}

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

export interface UserStreams {
  peerId: string
  streams: StreamWithURL[]
}

export interface StreamsState {
  localStreams: {
    [t in StreamType]?: LocalStream
  }
  streamsByPeerId: Record<string, UserStreams>
  metadataByPeerIdMid: Record<string, TrackMetadata>
  trackIdToPeerIdMid: Record<string, string>
  tracksByPeerIdMid: Record<string, TrackInfo>
}

interface TrackInfo {
  track: MediaStreamTrack
  mid: string
  association: TrackAssociation | undefined
}

interface TrackAssociation {
  streamId: string
  peerId: string
}

interface MidWithPeerId {
  mid: string
  streamId: string
  peerId: string
}

interface StreamIdPeerId {
  streamId: string
  peerId: string
}

/*
 * getPeerId returns the real user id from the metadata, if available, or
 * the peerId for the peer. In a normal P2P mesh network, each user will
 * have their own peer, which will correspond to their own peerId. In case of
 * an SFU, on peer connection from the server could provide tracks from
 * different users. That's why metadata is sent before each negotiation. The
 * metadata will contain a peerId paired with mid (transceiver).
 */
function getPeerId(
  state: StreamsState,
  payload: MidWithPeerId,
): StreamIdPeerId {
  const { mid } = payload
  const peerIdMid = getPeerIdMid(payload.peerId, mid)
  const metadata = state.metadataByPeerIdMid[peerIdMid]

  if (metadata) {
    debug(
      'streams getPeerId',
      payload.peerId, payload.streamId, metadata.peerId, metadata.streamId)
    return {
      peerId: metadata.peerId,
      streamId: metadata.streamId,
    }
  }

  debug('streams getPeerId', payload.peerId, payload.streamId)
  return {
    peerId: payload.peerId,
    streamId: payload.streamId,
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
  state: StreamsState, payload: RemoveLocalStreamPayload,
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

function removeTrack(
  state: StreamsState, track: MediaStreamTrack,
): StreamsState {
  debug('streams removeTrack: %o', track.id)
  const peerIdMid = state.trackIdToPeerIdMid[track.id]
  const t = state.tracksByPeerIdMid[peerIdMid]

  if (!t) {
    debug('streams removeTrack trackInfo not found', peerIdMid)
    return state
  }
  const {association} = t
  if (!association) {
    debug('streams removeTrack track not associated')
    return state
  }
  const {peerId, streamId} = association
  debug('streams removeTrack peerId: %s, streamId: %s', peerId, streamId)

  const userStreams = state.streamsByPeerId[peerId]
  if (!userStreams) {
    debug('streams removeTrack user streams not found')
    return state
  }

  let streams = userStreams.streams
  const s = streams.find(s => s.streamId === streamId)

  if (!s) {
    debug('streams removeTrack track stream not found', streamId)
    return state
  }

  debug('stream removeTrack: before tracks', s.stream.getTracks().length)
  s.stream.removeTrack(track)
  debug('stream removeTrack: after tracks', s.stream.getTracks().length)
  if (s.stream.getTracks().length === 0) {
    s.url && revokeObjectURL(s.url)
    streams = streams.filter(_s => _s !== s)
  }

  const tracksByPeerIdMid = {
    ...state.tracksByPeerIdMid,
    [peerIdMid]: {
      track,
      mid: t.mid,
      association: undefined,
    },
  }

  if (streams.length > 0) {
    return {
      ...state,
      streamsByPeerId: {
        ...state.streamsByPeerId,
        [peerId]: {
          ...userStreams,
          streams,
        },
      },
      tracksByPeerIdMid,
    }
  }

  debug('streams removeTrack removing user entry since no streams left')
  return {
    ...state,
    streamsByPeerId: omit(state.streamsByPeerId, [peerId]),
    tracksByPeerIdMid,
  }
}

function addTrack(
  state: StreamsState, payload: AddTrackPayload,
): StreamsState {
  debug('streams addTrack: %o', payload)
  const peerIdMid = getPeerIdMid(payload.peerId, payload.mid)
  const { peerId, streamId } = getPeerId(state, payload)
  const { track } = payload

  const userStreams = state.streamsByPeerId[peerId] || {
    streams: [],
    peerId,
  }

  const streams: StreamWithURL[] = userStreams.streams
  const existing = streams.find(s => s.streamId === streamId)

  if (existing) {
    debug('streams addTrack to existing stream')
    existing.stream.addTrack(track)
  } else {
    debug('streams addTrack to new stream')
    const stream = new MediaStream()
    stream.addTrack(track)
    streams.push({
      stream,
      streamId,
      url: safeCreateObjectURL(stream),
    })
  }

  return {
    ...state,
    streamsByPeerId: {
      ...state.streamsByPeerId,
      [peerId]: {
        ...userStreams,
        streams: [...streams],
      },
    },
    trackIdToPeerIdMid: {
      ...state.trackIdToPeerIdMid,
      [track.id]: peerIdMid,
    },
    tracksByPeerIdMid: {
      ...state.tracksByPeerIdMid,
      [peerIdMid]: {
        track,
        mid: payload.mid,
        association: {
          streamId,
          peerId,
        },
      },
    },
  }
}

export function unassociateUserTracks(
  state: StreamsState,
  payload: NicknameRemovePayload,
): StreamsState  {
  debug('streams unassociateUserTracks')
  const { peerId } = payload

  const userStreams = state.streamsByPeerId[peerId]
  if (!userStreams) {
    debug('streams unassociateUserTracks: user not found')
    return state
  }

  const tracksByPeerIdMid: Record<string, TrackInfo> = {}

  userStreams.streams.forEach(s => {
    s.stream.getTracks().forEach(track => {
      const peerIdMid = state.trackIdToPeerIdMid[track.id]
      tracksByPeerIdMid[peerIdMid] = {
        track,
        mid: state.tracksByPeerIdMid[peerIdMid].mid,
        association: undefined,
      }
      s.stream.removeTrack(track)
    })
  })

  const streamsByPeerId = omit(state.streamsByPeerId, [peerId])

  return {
    ...state,
    streamsByPeerId,
    tracksByPeerIdMid: {
      ...state.tracksByPeerIdMid,
      ...tracksByPeerIdMid,
    },
  }
}

function stopStream(s: StreamWithURL) {
  debug('streams stopStream()')
  s.stream.getTracks().forEach(track => {
    track.stop()
    track.onmute = null
    track.onunmute = null
  })
  s.url && revokeObjectURL(s.url)
}

function stopAllTracks(streams: StreamWithURL[]) {
  debug('streams stopAllTracks()')
  streams.forEach(s => stopStream(s))
}

function setMetadata(
  state: StreamsState,
  payload: MetadataPayload,
): StreamsState {
  debug('streams setMetadata: %o', payload)
  let newState = state
  const metadataByPeerIdMid = keyBy(
    payload.metadata,
    m => getPeerIdMid(payload.peerId, m.mid),
  )

  forEach(state.tracksByPeerIdMid, (t, peerIdMid) => {
    if (!metadataByPeerIdMid[peerIdMid] && t && t.association) {
      // remove any track the server has lost track of
      newState = removeTrack(newState, t.track)
    }
  })

  newState = {
    ...newState,
    metadataByPeerIdMid,
  }

  payload.metadata.forEach(m => {
    const { streamId, mid, peerId } = m
    const peerIdMid = getPeerIdMid(payload.peerId, mid)
    const t = state.tracksByPeerIdMid[peerIdMid]

    if (!t) {
      // this track hasn't been seen yet so there's nothing to do
      return
    }

    if (!t.association) {
      // add the unassociated track
      newState = addTrack(newState, {
        mid,
        streamId,
        track: t.track,
        peerId: payload.peerId,
      })
      return
    }

    const a = t.association
    if (a.streamId === streamId && a.peerId === peerId) {
      // track is associated with the right peerId / streamId
      return
    }

    newState = removeTrack(newState, t.track)
    newState = addTrack(newState, {
      mid,
      streamId: a.streamId,
      track: t.track,
      peerId: payload.peerId,
    })
  })

  return newState
}

function removePeer(
  state: StreamsState,
  payload: RemovePeerAction['payload'],
): StreamsState {
  debug('streams removePeer: %o', payload)
  let newState: StreamsState = state

  const keysToRemove = Object.keys(state.tracksByPeerIdMid)
  .filter(key => key.startsWith(payload.peerId + peerIdMidSeparator))

  const trackIdToPeerIdMid = state.trackIdToPeerIdMid

  keysToRemove.forEach(key => {
    const t = state.tracksByPeerIdMid[key]
    if (t.association) {
      newState = removeTrack(newState, t.track)
    }
    delete trackIdToPeerIdMid[t.track.id]
  })

  const tracksByPeerIdMid = omit(state.tracksByPeerIdMid, keysToRemove)

  return {
    ...newState,
    trackIdToPeerIdMid: {...trackIdToPeerIdMid},
    tracksByPeerIdMid,
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
    NicknameRemoveAction |
    RemovePeerAction |
    TracksMetadataAction,
): StreamsState {
  switch (action.type) {
    case STREAM_REMOVE:
      return removeLocalStream(state, action.payload)
    case STREAM_TRACK_ADD:
      return addTrack(state, action.payload)
    case STREAM_TRACK_REMOVE:
      return removeTrack(state, action.payload.track)
    case NICKNAME_REMOVE:
      return unassociateUserTracks(state, action.payload)
    case TRACKS_METADATA:
      return setMetadata(state, action.payload)
    case PEER_REMOVE:
      return removePeer(state, action.payload)
    case HANG_UP:
      forEach(state.localStreams, ls => stopStream(ls!))
      forEach(state.streamsByPeerId, us => stopAllTracks(us.streams))
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
