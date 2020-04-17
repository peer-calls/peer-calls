import _debug from 'debug'
import forEach from 'lodash/forEach'
import Peer, { SignalData } from 'simple-peer'
import { Decoder } from '../codec'
import * as constants from '../constants'
import { ClientSocket } from '../socket'
import { Dispatch, GetState } from '../store'
import { TextDecoder } from '../textcodec'
import { iceServers } from '../window'
import { addMessage } from './ChatActions'
import * as NotifyActions from './NotifyActions'
import * as StreamActions from './StreamActions'

const debug = _debug('peercalls')
const sdpDebug = _debug('peercalls:sdp')

export interface Peers {
  [id: string]: Peer.Instance
}

export interface PeerHandlerOptions {
  socket: ClientSocket
  user: { id: string }
  dispatch: Dispatch
  getState: GetState
}

class PeerHandler {
  socket: ClientSocket
  user: { id: string }
  dispatch: Dispatch
  getState: GetState
  decoder = new Decoder()
  textDecoder = new TextDecoder('utf-8')


  constructor (readonly options: PeerHandlerOptions) {
    this.socket = options.socket
    this.user = options.user
    this.dispatch = options.dispatch
    this.getState = options.getState
  }
  handleError = (err: Error) => {
    const { dispatch, getState, user } = this
    debug('peer: %s, error %s', user.id, err.stack)
    dispatch(NotifyActions.error('A peer connection error occurred'))
    const peer = getState().peers[user.id]
    peer && peer.destroy()
    dispatch(removePeer(user.id))
  }
  handleSignal = (signal: SignalData) => {
    const { socket, user } = this
    sdpDebug('local signal: %s, signal: %o', user.id, signal)

    const payload = { userId: user.id, signal }
    socket.emit('signal', payload)
  }
  handleConnect = () => {
    const { dispatch, user, getState } = this
    debug('peer: %s, connect', user.id)
    dispatch(NotifyActions.warning('Peer connection established'))

    const state = getState()
    const peer = state.peers[user.id]
    forEach(state.streams.localStreams, s => {
      // If the local user pressed join call before this peer has joined the
      // call, now is the time to share local media stream with the peer since
      // we no longer automatically send the stream to the peer.
      s!.stream.getTracks().forEach(track => {
        peer.addTrack(track, s!.stream)
      })
    })
  }
  handleTrack = (track: MediaStreamTrack, stream: MediaStream, mid: string) => {
    const { user, dispatch } = this
    const userId = user.id
    debug('peer: %s, track: %s, stream: %s, mid: %s',
          userId, track.id, stream.id, mid)
    // Listen to mute event to know when a track was removed
    // https://github.com/feross/simple-peer/issues/512
    track.onmute = () => {
      debug(
        'peer: %s, track mute (id: %s, stream.id: %s)',
        userId, track.id, stream.id)
      dispatch(StreamActions.removeTrack({ track }))
    }

    function addTrack() {
      debug(
        'peer: %s, track unmute (id: %s, stream.id: %s)',
        userId, track.id, stream.id)
      dispatch(StreamActions.addTrack({
        streamId: stream.id,
        mid,
        userId,
        track,
      }))
    }

    if (!track.muted) {
      addTrack()
    }
    track.onunmute = addTrack
  }
  handleData = (buffer: ArrayBuffer) => {
    const { dispatch, user } = this

    const dataContainer = this.decoder.decode(buffer)
    if (!dataContainer) {
      // not all chunks have been received yet
      return
    }

    const { data } = dataContainer
    const message = JSON.parse(this.textDecoder.decode(data))

    debug('peer: %s, message: %o', user.id, message)
    dispatch(addMessage(message))
  }
  handleClose = () => {
    const { dispatch, user } = this
    dispatch(NotifyActions.error('Peer connection closed'))
    dispatch(removePeer(user.id))
  }
}

export interface CreatePeerOptions {
  socket: ClientSocket
  user: { id: string }
  initiator: boolean
  stream?: MediaStream
}

/**
 * @param {Object} options
 * @param {Socket} options.socket
 * @param {User} options.user
 * @param {String} options.user.id
 * @param {Boolean} [options.initiator=false]
 * @param {MediaStream} [options.stream]
 */
export function createPeer (options: CreatePeerOptions) {
  const { socket, user, initiator, stream } = options

  return (dispatch: Dispatch, getState: GetState) => {
    const userId = user.id
    debug(
      'create peer: %s, hasStream: %s, initiator: %s',
      userId, !!stream, initiator)
    dispatch(NotifyActions.warning('Connecting to peer...'))

    const oldPeer = getState().peers[userId]
    if (oldPeer) {
      dispatch(NotifyActions.info('Cleaning up old connection...'))
      oldPeer.destroy()
      dispatch(removePeer(userId))
    }

    const peer = new Peer({
      initiator,
      config: { iceServers },
      channelName: constants.PEER_DATA_CHANNEL_NAME,
      // trickle: false,
      // Allow the peer to receive video, even if it's not sending stream:
      // https://github.com/feross/simple-peer/issues/95
      offerConstraints: {
        offerToReceiveAudio: true,
        offerToReceiveVideo: true,
      },
      stream,
    })

    const handler = new PeerHandler({
      socket,
      user,
      dispatch,
      getState,
    })

    peer.once(constants.PEER_EVENT_ERROR, handler.handleError)
    peer.once(constants.PEER_EVENT_CONNECT, handler.handleConnect)
    peer.once(constants.PEER_EVENT_CLOSE, handler.handleClose)
    peer.on(constants.PEER_EVENT_SIGNAL, handler.handleSignal)
    peer.on(constants.PEER_EVENT_TRACK, handler.handleTrack)
    peer.on(constants.PEER_EVENT_DATA, handler.handleData)

    dispatch(addPeer({ peer, userId }))
  }
}

export interface AddPeerParams {
  peer: Peer.Instance
  userId: string
}

export interface AddPeerAction {
  type: 'PEER_ADD'
  payload: AddPeerParams
}

export const addPeer = (payload: AddPeerParams): AddPeerAction => ({
  type: constants.PEER_ADD,
  payload,
})

export interface RemovePeerAction {
  type: 'PEER_REMOVE'
  payload: { userId: string }
}

export const removePeer = (userId: string): RemovePeerAction => ({
  type: constants.PEER_REMOVE,
  payload: { userId },
})

export type PeerAction =
  AddPeerAction |
  RemovePeerAction
