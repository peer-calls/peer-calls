import _debug from 'debug'
import { SocketEvent, TrackEventType } from '../SocketEvent'
import * as NotifyActions from '../actions/NotifyActions'
import * as PeerActions from '../actions/PeerActions'
import * as constants from '../constants'
import { ClientSocket } from '../socket'
import { Dispatch, GetState, Store } from '../store'
import { removeNickname, setNicknames } from './NicknameActions'
import { pubTrackEvent } from './StreamActions'

const debug = _debug('peercalls')
const sdpDebug = _debug('peercalls:sdp')

export interface SocketHandlerOptions {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: GetState
  peerId: string
}

class SocketHandler {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: GetState
  peerId: string

  constructor (options: SocketHandlerOptions) {
    this.socket = options.socket
    this.roomName = options.roomName
    this.stream = options.stream
    this.dispatch = options.dispatch
    this.getState = options.getState
    this.peerId = options.peerId
  }
  handleSignal = ({ peerId, signal }: SocketEvent['signal']) => {
    const { getState } = this
    const peer = getState().peers[peerId]
    sdpDebug('remote signal: peerId: %s, signal: %o', peerId, signal)
    if (!peer) return debug('user: %s, no peer found', peerId)
    peer.instance.signal(signal)
  }
  // One user has hung up
  handleHangUp = ({ peerId }: SocketEvent['hangUp']) => {
    const { dispatch } = this
    debug('socket hangUp, peerId: %s', peerId)
    dispatch(removeNickname({ peerId }))
  }
  handleUsers = ({ initiator, peerIds, nicknames }: SocketEvent['users']) => {
    const { socket, stream, dispatch, getState } = this
    debug('socket remote peerIds: %o', peerIds)

    this.dispatch(NotifyActions.info(
      'Connected users: {0}', Object.keys(nicknames).length))
    const { peers } = this.getState()
    debug('active peers: %o', Object.keys(peers))

    const isInitiator = initiator === this.peerId
    debug('isInitiator', isInitiator)

    dispatch(setNicknames(nicknames))

    peerIds
    .filter(peerId => !peers[peerId] && peerId !== this.peerId)
    .forEach(peerId => PeerActions.createPeer({
      socket,
      peer: {
        id: peerId,
      },
      initiator: isInitiator,
      stream,
    })(dispatch, getState))
  }
  handlePub = (pubTrack: SocketEvent['pubTrack']) => {
    const { dispatch } = this
    const { trackId, pubClientId, type } = pubTrack

    dispatch(pubTrackEvent(pubTrack))

    if (type == TrackEventType.Add) {
      this.socket.emit(constants.SOCKET_EVENT_SUB_TRACK, {
        trackId,
        type: TrackEventType.Sub,
        pubClientId,
      })
    }
  }
}

export interface HandshakeOptions {
  socket: ClientSocket
  store: Store
  roomName: string
  nickname: string
  peerId: string
  stream?: MediaStream
}

export function handshake (options: HandshakeOptions) {
  const { nickname, socket, roomName, stream, peerId, store } = options

  const handler = new SocketHandler({
    socket,
    roomName,
    stream,
    dispatch: store.dispatch,
    getState: store.getState,
    peerId,
  })

  // remove listeneres to make socket reusable
  removeEventListeners(socket)

  socket.on(constants.SOCKET_EVENT_SIGNAL, handler.handleSignal)
  socket.on(constants.SOCKET_EVENT_USERS, handler.handleUsers)
  socket.on(constants.SOCKET_EVENT_HANG_UP, handler.handleHangUp)
  socket.on(constants.SOCKET_EVENT_PUB_TRACK, handler.handlePub)

  debug('peerId: %s', peerId)
  socket.emit(constants.SOCKET_EVENT_READY, {
    room: roomName,
    nickname,
    peerId,
  })
}

export function removeEventListeners (socket: ClientSocket) {
  socket.removeAllListeners(constants.SOCKET_EVENT_SIGNAL)
  socket.removeAllListeners(constants.SOCKET_EVENT_USERS)
  socket.removeAllListeners(constants.SOCKET_EVENT_HANG_UP)
  socket.removeAllListeners(constants.SOCKET_EVENT_PUB_TRACK)
}
