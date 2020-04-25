// import _debug from 'debug'
import forEach from 'lodash/forEach'
import keyBy from 'lodash/keyBy'
import omit from 'lodash/omit'
import { HangUpAction } from '../actions/CallActions'
import { MediaStreamAction } from '../actions/MediaActions'
import { RemovePeerAction } from '../actions/PeerActions'
import { AddTrackPayload, RemoveLocalStreamPayload, RemoveTrackPayload, StreamAction, StreamType, TracksMetadataAction, AddLocalStreamPayload } from '../actions/StreamActions'
import { HANG_UP, MEDIA_STREAM, PEER_REMOVE, STREAM_REMOVE, STREAM_TRACK_ADD, STREAM_TRACK_REMOVE, NICKNAME_REMOVE, TRACKS_METADATA } from '../constants'
import { createObjectURL, revokeObjectURL } from '../window'
import { NicknameRemoveAction, NicknameRemovePayload } from '../actions/NicknameActions'
import { TrackMetadata, MetadataPayload } from '../../shared'
import { MediaStream } from '../window'

// const debug = _debug('peercalls')
const defaultState = Object.freeze({
  localStreams: {},
  streamsByUserId: {},
  metadataByPeerIdMid: {},
  trackIdToPeerIdMid: {},
  tracksByPeerIdMid: {},
})

const peerIdMidSeparator = '::'

function getPeerIdMid(userId: string, mid: string): string {
  return userId + peerIdMidSeparator + mid
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
}

export interface UserStreams {
  userId: string
  streams: StreamWithURL[]
}

export interface StreamsState {
  localStreams: {
    [t in StreamType]?: LocalStream
  }
  streamsByUserId: Record<string, UserStreams>
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
  userId: string
}

interface MidWithUserId {
  mid: string
  streamId: string
  userId: string
}

interface StreamIdUserId {
  streamId: string
  userId: string
}

/*
 * getUserId returns the real user id from the metadata, if available, or
 * the userId for the peer. In a normal P2P mesh network, each user will
 * have their own peer, which will correspond to their own userId. In case of
 * an SFU, on peer connection from the server could provide tracks from
 * different users. That's why metadata is sent before each negotiation. The
 * metadata will contain a userId paired with mid (transceiver).
 */
function getUserId(
  state: StreamsState,
  payload: MidWithUserId,
): StreamIdUserId {
  const { mid } = payload
  const peerIdMid = getPeerIdMid(payload.userId, mid)
  const metadata = state.metadataByPeerIdMid[peerIdMid]

  if (metadata) {
    return {
      userId: metadata.userId,
      streamId: metadata.streamId,
    }
  }

  return {
    userId: payload.userId,
    streamId: payload.streamId,
  }
}

function addLocalStream (
  state: StreamsState, payload: AddLocalStreamPayload,
): StreamsState {
  const { stream } = payload

  const streamWithURL: LocalStream = {
    stream: payload.stream,
    streamId: payload.stream.id,
    type: payload.type,
    url: safeCreateObjectURL(stream),
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
  state: StreamsState, payload: RemoveTrackPayload,
): StreamsState {
  const peerIdMid = getPeerIdMid(payload.userId, payload.mid)
  const { userId, streamId } = getUserId(state, payload)
  const { track } = payload

  const userStreams = state.streamsByUserId[userId]
  if (!userStreams) {
    console.log('userId not found', userId)
    return state
  }

  let streams = userStreams.streams
  const s = streams.find(s => s.streamId = streamId)

  if (!s) {
    return state
  }

  s.stream.removeTrack(track)
  if (s.stream.getTracks().length === 0) {
    s.url && revokeObjectURL(s.url)
    streams = streams.filter(_s => _s !== s)
  }

  const tracksByPeerIdMid = {
    ...state.tracksByPeerIdMid,
    [peerIdMid]: {
      track,
      mid: payload.mid,
      association: undefined,
    },
  }

  if (streams.length > 0) {
    return {
      ...state,
      streamsByUserId: {
        ...state.streamsByUserId,
        [userId]: {
          ...userStreams,
          streams,
        },
      },
      tracksByPeerIdMid,
    }
  }

  return {
    ...state,
    streamsByUserId: omit(state.streamsByUserId, [userId]),
    tracksByPeerIdMid,
  }
}

function addTrack(
  state: StreamsState, payload: AddTrackPayload,
): StreamsState {
  const peerIdMid = getPeerIdMid(payload.userId, payload.mid)
  const { userId, streamId } = getUserId(state, payload)
  const { track } = payload

  const userStreams = state.streamsByUserId[userId] || {
    streams: [],
    userId,
  }

  const streams: StreamWithURL[] = userStreams.streams
  const existing = streams.find(s => s.streamId === streamId)

  if (existing) {
    existing.stream.addTrack(track)
  } else {
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
    streamsByUserId: {
      ...state.streamsByUserId,
      [userId]: {
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
          userId,
        },
      },
    },
  }
}

export function unassociateUserTracks(
  state: StreamsState,
  payload: NicknameRemovePayload,
): StreamsState  {
  const { userId } = payload

  const userStreams = state.streamsByUserId[userId]
  if (!userStreams) {
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

  const streamsByUserId = omit(state.streamsByUserId, [userId])

  return {
    ...state,
    streamsByUserId,
    tracksByPeerIdMid: {
      ...state.tracksByPeerIdMid,
      ...tracksByPeerIdMid,
    },
  }
}

function stopStream(s: StreamWithURL) {
  s.stream.getTracks().forEach(track => {
    track.stop()
    track.onmute = null
    track.onunmute = null
  })
  s.url && revokeObjectURL(s.url)
}

function stopAllTracks(streams: StreamWithURL[]) {
  streams.forEach(s => stopStream(s))
}

function setMetadata(
  state: StreamsState,
  payload: MetadataPayload,
): StreamsState {

  const oldMetadata = state.metadataByPeerIdMid

  let newState = state
  const newMetadata = keyBy(
    payload.metadata,
    m => getPeerIdMid(payload.userId, m.mid),
  )

  const omitOldKeys: string[] = []
  forEach(oldMetadata, m => {
    const  { mid } = m
    const peerIdMid = getPeerIdMid(payload.userId, mid)
    const t = state.tracksByPeerIdMid[peerIdMid]

    if (!newMetadata[peerIdMid] && t && t.association) {
      // remove any track the server has lost track of
      newState = removeTrack(newState, {
        mid,
        streamId: t.association.streamId,
        track: t.track,
        userId: t.association.userId,
      })
      omitOldKeys.push(peerIdMid)
    }
  })

  const metadataByPeerIdMid = {
    ...omit(oldMetadata, omitOldKeys),
    ...newMetadata,
  }

  payload.metadata.forEach(m => {
    const { streamId, mid, userId } = m
    const peerIdMid = getPeerIdMid(payload.userId, mid)
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
        userId,
      })
      return
    }

    const a = t.association
    if (a.streamId === streamId && a.userId === userId) {
      // track is associated with the right userId / streamId
      return
    }

    newState = removeTrack(newState, {
      mid,
      streamId: a.streamId,
      track: t.track,
      userId: a.userId,
    })
    newState = addTrack(newState, {
      mid,
      streamId,
      track: t.track,
      userId,
    })
  })

  return {
    ...newState,
    metadataByPeerIdMid,
  }
}

function removePeer(
  state: StreamsState,
  action: RemovePeerAction['payload'],
): StreamsState {
  let newState: StreamsState = state

  const keysToRemove = Object.keys(state.tracksByPeerIdMid)
  .filter(key => key.startsWith(action.userId + peerIdMidSeparator))

  const trackIdToPeerIdMid = state.trackIdToPeerIdMid

  keysToRemove.forEach(key => {
    const t = state.tracksByPeerIdMid[key]
    delete trackIdToPeerIdMid[t.track.id]
    if (t.association) {
      newState = removeTrack(newState, {
        mid: t.mid,
        streamId: t.association.streamId,
        track: t.track,
        userId: t.association.userId,
      })
    }
  })

  const tracksByPeerIdMid = omit(state.tracksByPeerIdMid, keysToRemove)

  return {
    ...newState,
    trackIdToPeerIdMid,
    tracksByPeerIdMid,
  }
}

export default function streams(
  state: StreamsState = defaultState,
  action:
    StreamAction |
    MediaStreamAction |
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
      return removeTrack(state, action.payload)
    case NICKNAME_REMOVE:
      return unassociateUserTracks(state, action.payload)
    case TRACKS_METADATA:
      return setMetadata(state, action.payload)
    case PEER_REMOVE:
      return removePeer(state, action.payload)
    case HANG_UP:
      forEach(state.localStreams, ls => stopStream(ls!))
      forEach(state.streamsByUserId, us => stopAllTracks(us.streams))
      return defaultState
    case MEDIA_STREAM:
      if (action.status === 'resolved') {
        return addLocalStream(state, action.payload)
      } else {
        return state
      }
    default:
      return state
  }
}
