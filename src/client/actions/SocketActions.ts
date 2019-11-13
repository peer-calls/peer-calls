import * as NotifyActions from '../actions/NotifyActions'
import * as PeerActions from '../actions/PeerActions'
import * as constants from '../constants'
import _ from 'underscore'
import _debug from 'debug'
import { Dispatch } from 'redux'
import { SignalData } from 'simple-peer'

const debug = _debug('peercalls')

export interface SocketHandlerOptions {
  socket: SocketIOClient.Socket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: PeerActions.GetState
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
  socket: SocketIOClient.Socket
  roomName: string
  stream?: MediaStream
  dispatch: Dispatch
  getState: PeerActions.GetState

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
    NotifyActions.info('Connected users: {0}', users.length)(dispatch)
    const { peers } = getState()

    users
    .filter(user => !peers[user.id] && user.id !== socket.id)
    .forEach(user => PeerActions.createPeer({
      socket,
      user,
      initiator,
      stream,
    })(dispatch, getState))

    const newUsersMap = _.indexBy(users, 'id')
    _.keys(peers)
    .filter(id => !newUsersMap[id])
    .forEach(id => peers[id].destroy())
  }
}

export interface HandshakeOptions {
  socket: SocketIOClient.Socket
  roomName: string
  stream?: MediaStream
}

export function handshake (options: HandshakeOptions) {
  const { socket, roomName, stream } = options

  return (dispatch: Dispatch, getState: PeerActions.GetState) => {
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
    NotifyActions.info('Ready for connections')(dispatch)
    socket.emit('ready', roomName)
  }
}
