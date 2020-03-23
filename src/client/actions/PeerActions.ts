import * as ChatActions from './ChatActions'
import * as NicknameActions from './NicknameActions'
import * as NotifyActions from './NotifyActions'
import * as StreamActions from './StreamActions'
import * as constants from '../constants'
import Peer, { SignalData } from 'simple-peer'
import forEach from 'lodash/forEach'
import _debug from 'debug'
import { iceServers } from '../window'
import { Dispatch, GetState } from '../store'
import { ClientSocket } from '../socket'
import { getNickname } from '../nickname'

const debug = _debug('peercalls')

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
    debug('peer: %s, signal: %o', user.id, signal)

    const payload = { userId: user.id, signal }
    socket.emit('signal', payload)
  }
  handleConnect = () => {
    const { dispatch, user, getState } = this
    debug('peer: %s, connect', user.id)
    dispatch(NotifyActions.warning('Peer connection established'))

    const state = getState()
    const peer = state.peers[user.id]
    const localStream = state.streams[constants.ME]
    localStream && localStream.streams.forEach(s => {
      // If the local user pressed join call before this peer has joined the
      // call, now is the time to share local media stream with the peer since
      // we no longer automatically send the stream to the peer.
      s.stream.getTracks().forEach(track => {
        peer.addTrack(track, s.stream)
      })
    })
    const nickname = state.nicknames[constants.ME]
    if (nickname) {
      sendData(peer, {
        payload: {nickname},
        type: 'nickname',
      })
    }
  }
  handleTrack = (track: MediaStreamTrack, stream: MediaStream) => {
    const { user, dispatch } = this
    const userId = user.id
    debug('peer: %s, track', userId)
    // Listen to mute event to know when a track was removed
    // https://github.com/feross/simple-peer/issues/512
    track.onmute = () => {
      debug('peer: %s, track muted', userId)
      dispatch(StreamActions.removeTrack({
        userId,
        stream,
        track,
      }))
    }
    dispatch(StreamActions.addStream({
      userId,
      stream,
    }))
  }
  handleData = (buffer: ArrayBuffer) => {
    const { dispatch, getState, user } = this
    const state = getState()
    const message = JSON.parse(new window.TextDecoder('utf-8').decode(buffer))
    debug('peer: %s, message: %o', user.id, message)
    switch (message.type) {
      case 'file':
        dispatch(ChatActions.addMessage({
          userId: user.id,
          message: message.payload.name,
          timestamp: new Date().toLocaleString(),
          image: message.payload.data,
        }))
        break
      case 'nickname':
        dispatch(ChatActions.addMessage({
          userId: constants.PEERCALLS,
          message: 'User ' + getNickname(state.nicknames, user.id) +
            ' is now known as ' + (message.payload.nickname || user.id),
          timestamp: new Date().toLocaleString(),
          image: undefined,
        }))
        dispatch(NicknameActions.setNickname({
          userId: user.id,
          nickname: message.payload.nickname,
        }))
        break
      default:
        dispatch(ChatActions.addMessage({
          userId: user.id,
          message: message.payload,
          timestamp: new Date().toLocaleString(),
          image: undefined,
        }))
    }
  }
  handleClose = () => {
    const { dispatch, user, getState } = this
    dispatch(NotifyActions.error('Peer connection closed'))
    const state = getState()
    const userStreams = state.streams[user.id]
    userStreams && userStreams.streams.forEach(s => {
      dispatch(StreamActions.removeStream(user.id, s.stream))
    })
    dispatch(removePeer(user.id))
  }
}

export interface CreatePeerOptions {
  socket: ClientSocket
  user: { id: string }
  initiator: string
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
    debug('create peer: %s, stream:', userId, stream)
    dispatch(NotifyActions.warning('Connecting to peer...'))

    const oldPeer = getState().peers[userId]
    if (oldPeer) {
      dispatch(NotifyActions.info('Cleaning up old connection...'))
      oldPeer.destroy()
      dispatch(removePeer(userId))
    }

    const peer = new Peer({
      initiator: userId === initiator,
      config: { iceServers },
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

export interface DestroyPeersAction {
  type: 'PEERS_DESTROY'
}

export const destroyPeers = (): DestroyPeersAction => ({
  type: constants.PEERS_DESTROY,
})

export type PeerAction =
  AddPeerAction |
  RemovePeerAction |
  DestroyPeersAction

export interface TextMessage {
  type: 'text'
  payload: string
}

export interface Base64File {
  name: string
  size: number
  type: string
  data: string
}

export interface FileMessage {
  type: 'file'
  payload: Base64File
}

export interface NicknameMessage {
  type: 'nickname'
  payload: {
    nickname: string
  }
}

export type Message = TextMessage | FileMessage | NicknameMessage

function sendData(peer: Peer.Instance, message: Message) {
  peer.send(JSON.stringify(message))
}

export const sendMessage = (message: Message) =>
(dispatch: Dispatch, getState: GetState) => {
  const { peers } = getState()
  debug('Sending message type: %s to %s peers.',
    message.type, Object.keys(peers).length)
  switch (message.type) {
    case 'file':
      dispatch(ChatActions.addMessage({
        userId: constants.ME,
        message: 'Send file: "' +
          message.payload.name + '" to all peers',
        timestamp: new Date().toLocaleString(),
        image: message.payload.data,
      }))
      break
    case 'nickname':
      dispatch(ChatActions.addMessage({
        userId: constants.PEERCALLS,
        message: 'You are now known as: ' + message.payload.nickname,
        timestamp: new Date().toLocaleString(),
        image: undefined,
      }))
      dispatch(NicknameActions.setNickname({
        userId: constants.ME,
        nickname: message.payload.nickname,
      }))
      window.localStorage &&
        (window.localStorage.nickname = message.payload.nickname)
      break
    default:
      dispatch(ChatActions.addMessage({
        userId: constants.ME,
        message: message.payload,
        timestamp: new Date().toLocaleString(),
        image: undefined,
      }))
  }
  forEach(peers, (peer, userId) => {
    sendData(peer, message)
  })
}

export const sendFile = (file: File) =>
async (dispatch: Dispatch, getState: GetState) => {
  const { name, size, type } = file
  if (!window.FileReader) {
    dispatch(NotifyActions.error('File API is not supported by your browser'))
    return
  }
  const reader = new window.FileReader()
  const base64File = await new Promise<Base64File>(resolve => {
    reader.addEventListener('load', () => {
      resolve({
        name,
        size,
        type,
        data: reader.result as string,
      })
    })
    reader.readAsDataURL(file)
  })

  sendMessage({ payload: base64File, type: 'file' })(dispatch, getState)
}
