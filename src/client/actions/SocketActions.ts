import * as NotifyActions from '../actions/NotifyActions'
import * as PeerActions from '../actions/PeerActions'
import * as constants from '../constants'
import _debug from 'debug'
import { Dispatch, GetState } from '../store'
import { ClientSocket } from '../socket'
import { SocketEvent } from '../../shared'

const debug = _debug('peercalls')

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
    // debug('socket signal, userId: %s, signal: %o', userId, signal);
    if (!peer) return debug('user: %s, no peer found', userId)
    peer.signal(signal)
  }
  handleUsers = ({ initiator, users }: SocketEvent['users']) => {
    const { socket, stream, dispatch, getState } = this
    debug('socket users: %o', users)
    this.dispatch(NotifyActions.info('Connected users: {0}', users.length))
    const { peers } = this.getState()
    debug('active peers: %o', Object.keys(peers))

    users
    .filter(
      user =>
      user.userId && !peers[user.userId] && user.userId !== this.userId)
    .forEach(user => PeerActions.createPeer({
      socket,
      user: {
        // users without id should be filtered out
        id: user.userId!,
      },
      initiator,
      stream,
    })(dispatch, getState))
  }
}

export interface HandshakeOptions {
  socket: ClientSocket
  roomName: string
  userId: string
  stream?: MediaStream
}

export function handshake (options: HandshakeOptions) {
  const { socket, roomName, stream, userId } = options

  return (dispatch: Dispatch, getState: GetState) => {
    const handler = new SocketHandler({
      socket,
      roomName,
      stream,
      dispatch,
      getState,
      userId,
    })

    // remove listeneres to make seocket reusable
    socket.removeListener(constants.SOCKET_EVENT_SIGNAL, handler.handleSignal)
    socket.removeListener(constants.SOCKET_EVENT_USERS, handler.handleUsers)

    socket.on(constants.SOCKET_EVENT_SIGNAL, handler.handleSignal)
    socket.on(constants.SOCKET_EVENT_USERS, handler.handleUsers)

    debug('userId: %s', userId)
    debug('emit ready for room: %s', roomName)
    dispatch(NotifyActions.info('Ready for connections'))
    socket.emit('ready', {
      room: roomName,
      userId,
    })
  }
}
