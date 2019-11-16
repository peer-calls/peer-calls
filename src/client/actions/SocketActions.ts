import * as NotifyActions from '../actions/NotifyActions'
import * as PeerActions from '../actions/PeerActions'
import * as constants from '../constants'
import keyBy from 'lodash/keyBy'
import _debug from 'debug'
import { SignalData } from 'simple-peer'
import { Dispatch, GetState } from '../store'
import { ClientSocket } from '../socket'

const debug = _debug('peercalls')

export interface SocketHandlerOptions {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: GetState
}

export interface SignalOptions {
  signal: SignalData
  userId: string
}

export interface UsersOptions {
  initiator: string
  users: Array<{ id: string }>
}

class SocketHandler {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: GetState

  constructor (options: SocketHandlerOptions) {
    this.socket = options.socket
    this.roomName = options.roomName
    this.stream = options.stream
    this.dispatch = options.dispatch
    this.getState = options.getState
  }
  handleSignal = ({ userId, signal }: SignalOptions) => {
    const { getState } = this
    const peer = getState().peers[userId]
    // debug('socket signal, userId: %s, signal: %o', userId, signal);
    if (!peer) return debug('user: %s, no peer found', userId)
    peer.signal(signal)
  }
  handleUsers = ({ initiator, users }: UsersOptions) => {
    const { socket, stream, dispatch, getState } = this
    debug('socket users: %o', users)
    this.dispatch(NotifyActions.info('Connected users: {0}', users.length))
    const { peers } = this.getState()

    users
    .filter(user => !peers[user.id] && user.id !== socket.id)
    .forEach(user => PeerActions.createPeer({
      socket,
      user,
      initiator,
      stream,
    })(dispatch, getState))

    const newUsersMap = keyBy(users, 'id')
    Object.keys(peers)
    .filter(id => !newUsersMap[id])
    .forEach(id => peers[id].destroy())
  }
}

export interface HandshakeOptions {
  socket: ClientSocket
  roomName: string
  stream?: MediaStream
}

export function handshake (options: HandshakeOptions) {
  const { socket, roomName, stream } = options

  return (dispatch: Dispatch, getState: GetState) => {
    const handler = new SocketHandler({
      socket,
      roomName,
      stream,
      dispatch,
      getState,
    })

    socket.on(constants.SOCKET_EVENT_SIGNAL, handler.handleSignal)
    socket.on(constants.SOCKET_EVENT_USERS, handler.handleUsers)

    debug('socket.id: %s', socket.id)
    debug('emit ready for room: %s', roomName)
    dispatch(NotifyActions.info('Ready for connections'))
    socket.emit('ready', roomName)
  }
}
