import _debug from 'debug'
import { MetadataPayload, SocketEvent } from '../SocketEvent'
import * as NotifyActions from '../actions/NotifyActions'
import * as PeerActions from '../actions/PeerActions'
import * as constants from '../constants'
import { ClientSocket } from '../socket'
import { Dispatch, GetState, Store } from '../store'
import { removeNickname, setNicknames } from './NicknameActions'
import { tracksMetadata } from './StreamActions'

const debug = _debug('peercalls')
const sdpDebug = _debug('peercalls:sdp')

export interface SocketHandlerOptions {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: GetState
  userId: string
}

class SocketHandler {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: GetState
  userId: string

  constructor (options: SocketHandlerOptions) {
    this.socket = options.socket
    this.roomName = options.roomName
    this.stream = options.stream
    this.dispatch = options.dispatch
    this.getState = options.getState
    this.userId = options.userId
  }
  handleSignal = ({ userId, signal }: SocketEvent['signal']) => {
    const { getState } = this
    const peer = getState().peers[userId]
    sdpDebug('remote signal: userId: %s, signal: %o', userId, signal)
    if (!peer) return debug('user: %s, no peer found', userId)
    peer.signal(signal)
  }
  // One user has hung up
  handleHangUp = ({ userId }: SocketEvent['hangUp']) => {
    const { dispatch } = this
    debug('socket hangUp, userId: %s', userId)
    dispatch(removeNickname({ userId }))
  }
  handleMetadata = (payload: MetadataPayload) => {
    const { dispatch } = this
    debug('metadata', payload)
    dispatch(tracksMetadata(payload))
  }
  handleUsers = ({ initiator, peerIds, nicknames }: SocketEvent['users']) => {
    const { socket, stream, dispatch, getState } = this
    debug('socket remote peerIds: %o', peerIds)

    this.dispatch(NotifyActions.info(
      'Connected users: {0}', Object.keys(nicknames).length))
    const { peers } = this.getState()
    debug('active peers: %o', Object.keys(peers))

    const isInitiator = initiator === this.userId
    debug('isInitiator', isInitiator)

    dispatch(setNicknames(nicknames))

    peerIds
    .filter(peerId => !peers[peerId] && peerId !== this.userId)
    .forEach(peerId => PeerActions.createPeer({
      socket,
      user: {
        id: peerId,
      },
      initiator: isInitiator,
      stream,
    })(dispatch, getState))
  }
}

export interface HandshakeOptions {
  socket: ClientSocket
  store: Store
  roomName: string
  nickname: string
  userId: string
  stream?: MediaStream
}

export function handshake (options: HandshakeOptions) {
  const { nickname, socket, roomName, stream, userId, store } = options

  const handler = new SocketHandler({
    socket,
    roomName,
    stream,
    dispatch: store.dispatch,
    getState: store.getState,
    userId,
  })

  // remove listeneres to make socket reusable
  removeEventListeners(socket)

  socket.on(constants.SOCKET_EVENT_METADATA, handler.handleMetadata)
  socket.on(constants.SOCKET_EVENT_SIGNAL, handler.handleSignal)
  socket.on(constants.SOCKET_EVENT_USERS, handler.handleUsers)
  socket.on(constants.SOCKET_EVENT_HANG_UP, handler.handleHangUp)

  debug('userId: %s', userId)
  socket.emit(constants.SOCKET_EVENT_READY, {
    room: roomName,
    nickname,
    userId,
  })
}

export function removeEventListeners (socket: ClientSocket) {
  socket.removeAllListeners(constants.SOCKET_EVENT_METADATA)
  socket.removeAllListeners(constants.SOCKET_EVENT_SIGNAL)
  socket.removeAllListeners(constants.SOCKET_EVENT_USERS)
  socket.removeAllListeners(constants.SOCKET_EVENT_HANG_UP)
}
