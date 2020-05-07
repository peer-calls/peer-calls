import * as ChatActions from './ChatActions'
import * as NotifyActions from './NotifyActions'
import * as StreamActions from './StreamActions'
import * as constants from '../constants'
import Peer, { SignalData } from 'simple-peer'
import forEach from 'lodash/forEach'
import _debug from 'debug'
import { iceServers, userId } from '../window'
import { Dispatch, GetState } from '../store'
import { ClientSocket } from '../socket'
import { Encoder, Decoder } from '../codec'

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
  decoder?: Decoder

  constructor (readonly options: PeerHandlerOptions) {
    this.socket = options.socket
    this.user = options.user
    this.dispatch = options.dispatch
    this.getState = options.getState
  }
  getDecoder(): Decoder {
    if (this.decoder) {
      return this.decoder
    }
    this.decoder = new Decoder()
    return this.decoder
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
    track.onunmute = () => {
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
  }
  handleData = (buffer: ArrayBuffer) => {
    console.log('handleData', buffer)
    const { dispatch, user } = this

    const dataContainer = this.getDecoder().decode(buffer)
    if (!dataContainer) {
      console.log('data - waiting for other chunks')
      // not all chunks have been received yet
      return
    }

    const { senderId, data } = dataContainer
    const message = JSON.parse(new TextDecoder('utf-8').decode(data))

    debug('peer: %s, message: %o', user.id, message)
    switch (message.type) {
      case 'file':
        dispatch(ChatActions.addMessage({
          userId: senderId,
          message: message.payload.name,
          timestamp: new Date().toLocaleString(),
          image: message.payload.data,
        }))
        break
      default:
        dispatch(ChatActions.addMessage({
          userId: senderId,
          message: message.payload,
          timestamp: new Date().toLocaleString(),
          image: undefined,
        }))
    }
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
      trickle: false,
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

export type Message = TextMessage | FileMessage

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
    default:
      dispatch(ChatActions.addMessage({
        userId: constants.ME,
        message: message.payload,
        timestamp: new Date().toLocaleString(),
        image: undefined,
      }))
  }

  const encoder = new Encoder()
  const chunks = encoder.encode({
    senderId: userId,
    data: new TextEncoder().encode(JSON.stringify(message)),
  })
  chunks.forEach(chunk => {
    console.log('sending chunk', chunk)
    forEach(peers, (peer, userId) => {
      peer.send(chunk)
    })
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
