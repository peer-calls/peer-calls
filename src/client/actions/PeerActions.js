import * as NotifyActions from '../actions/NotifyActions.js'
import * as StreamActions from '../actions/StreamActions.js'
import * as constants from '../constants.js'
import Peer from 'simple-peer'
import _ from 'underscore'
import _debug from 'debug'
import { play, iceServers } from '../window.js'

const debug = _debug('peercalls')

class PeerHandler {
  constructor ({ socket, user, dispatch, getState }) {
    this.socket = socket
    this.user = user
    this.dispatch = dispatch
    this.getState = getState
  }
  handleError = err => {
    const { dispatch, getState, user } = this
    debug('peer: %s, error %s', user.id, err.stack)
    dispatch(NotifyActions.error('A peer connection error occurred'))
    const peer = getState().peers[user.id]
    peer && peer.destroy()
    dispatch(removePeer(user.id))
  }
  handleSignal = signal => {
    const { socket, user } = this
    debug('peer: %s, signal: %o', user.id, signal)

    const payload = { userId: user.id, signal }
    socket.emit('signal', payload)
  }
  handleConnect = () => {
    const { dispatch, user } = this
    debug('peer: %s, connect', user.id)
    dispatch(NotifyActions.warning('Peer connection established'))
    play()
  }
  handleStream = stream => {
    const { user, dispatch } = this
    debug('peer: %s, stream', user.id)
    dispatch(StreamActions.addStream({
      userId: user.id,
      stream
    }))
  }
  handleData = object => {
    const { dispatch, user } = this
    object = JSON.parse(new window.TextDecoder('utf-8').decode(object))
    debug('peer: %s, message: %o', user.id, object)
    const message = user.id + ': ' + object.message
    dispatch(NotifyActions.info(message))
  }
  handleClose = () => {
    const { dispatch, user } = this
    debug('peer: %s, close', user.id)
    dispatch(NotifyActions.error('Peer connection closed'))
    dispatch(StreamActions.removeStream(user.id))
    dispatch(removePeer(user.id))
  }
}

/**
 * @param {Object} options
 * @param {Socket} options.socket
 * @param {User} options.user
 * @param {String} options.user.id
 * @param {Boolean} [options.initiator=false]
 * @param {MediaStream} [options.stream]
 */
export function createPeer ({ socket, user, initiator, stream }) {
  return (dispatch, getState) => {
    const userId = user.id
    debug('create peer: %s, stream:', userId, stream)
    dispatch(NotifyActions.warning('Connecting to peer...'))

    const oldPeer = getState().peers[userId]
    if (oldPeer) {
      dispatch(NotifyActions.info('Cleaning up old connection...'))
      oldPeer.destroy()
      dispatch(removePeer(userId))
    }

    const peer = new Peer({
      initiator: socket.id === initiator,
      config: { iceServers },
      // Allow the peer to receive video, even if it's not sending stream:
      // https://github.com/feross/simple-peer/issues/95
      offerConstraints: {
        offerToReceiveAudio: true,
        offerToReceiveVideo: true
      },
      stream
    })

    const handler = new PeerHandler({
      socket,
      user,
      dispatch,
      getState
    })

    peer.once(constants.PEER_EVENT_ERROR, handler.handleError)
    peer.once(constants.PEER_EVENT_CONNECT, handler.handleConnect)
    peer.once(constants.PEER_EVENT_CLOSE, handler.handleClose)
    peer.on(constants.PEER_EVENT_SIGNAL, handler.handleSignal)
    peer.on(constants.PEER_EVENT_STREAM, handler.handleStream)
    peer.on(constants.PEER_EVENT_DATA, handler.handleData)

    dispatch(addPeer({ peer, userId }))
  }
}

export const addPeer = ({ peer, userId }) => ({
  type: constants.PEER_ADD,
  payload: { peer, userId }
})

export const removePeer = userId => ({
  type: constants.PEER_REMOVE,
  payload: { userId }
})

export const destroyPeers = () => ({
  type: constants.PEERS_DESTROY
})

export const sendMessage = message => (dispatch, getState) => {
  message = JSON.stringify({ message })
  const { peers } = getState()
  _.each(peers, peer => peer.send(message))
}
