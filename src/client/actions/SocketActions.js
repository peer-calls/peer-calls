import * as ChatActions from '../actions/ChatActions.js'
import * as NotifyActions from '../actions/NotifyActions.js'
import * as PeerActions from '../actions/PeerActions.js'
import * as constants from '../constants.js'
import _ from 'underscore'
import _debug from 'debug'

const debug = _debug('peercalls')

class SocketHandler {
  constructor ({ socket, roomName, stream, dispatch, getState }) {
    this.socket = socket
    this.roomName = roomName
    this.stream = stream
    this.dispatch = dispatch
    this.getState = getState
  }
  handleSignal = ({ userId, signal }) => {
    const { getState } = this
    const peer = getState().peers[userId]
    // debug('socket signal, userId: %s, signal: %o', userId, signal);
    if (!peer) return debug('user: %s, no peer found', userId)
    peer.signal(signal)
  }
  handleUsers = ({ initiator, users }) => {
    const { socket, stream, dispatch, getState } = this
    debug('socket users: %o', users)
    dispatch(NotifyActions.info('Connected users: {0}', users.length))
    const { peers } = getState()

    users
    .filter(user => !peers[user.id] && user.id !== socket.id)
    .forEach(user => dispatch(PeerActions.createPeer({
      socket,
      user,
      initiator,
      stream
    })))

    let newUsersMap = _.indexBy(users, 'id')
    _.keys(peers)
    .filter(id => !newUsersMap[id])
    .forEach(id => peers[id].destroy())
  }
  handleMessages = (messages) => {
    const { dispatch } = this
    debug('socket messages: %o', messages)
    dispatch(ChatActions.loadHistory(messages))
  }
  handleNewMessage = (payload) => {
    const { dispatch } = this
    debug('socket message: %o', payload)
    dispatch(ChatActions.addMessage(payload))
  }
}

export function handshake ({ socket, roomName, stream }) {
  return (dispatch, getState) => {
    const handler = new SocketHandler({
      socket,
      roomName,
      stream,
      dispatch,
      getState
    })

    socket.on(constants.SOCKET_EVENT_SIGNAL, handler.handleSignal)
    socket.on(constants.SOCKET_EVENT_USERS, handler.handleUsers)
    socket.on(constants.SOCKET_EVENT_MESSAGES, handler.handleMessages)
    socket.on(constants.SOCKET_EVENT_NEW_MESSAGE, handler.handleNewMessage)

    debug('socket.id: %s', socket.id)
    debug('emit ready for room: %s', roomName)
    dispatch(NotifyActions.info('Ready for connections'))
    socket.emit('ready', roomName)
  }
}
