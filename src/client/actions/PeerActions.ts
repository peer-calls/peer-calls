import _debug from 'debug'
import Peer, { SignalData } from 'simple-peer'
import { Decoder } from '../codec'
import * as constants from '../constants'
import { ClientSocket } from '../socket'
import { Dispatch, GetState } from '../store'
import { TextDecoder } from '../textcodec'
import { config } from '../window'
import { addMessage } from './ChatActions'
import * as NotifyActions from './NotifyActions'
import * as StreamActions from './StreamActions'

const { peerConfig } = config

const debug = _debug('peercalls')
const sdpDebug = _debug('peercalls:sdp')

export interface Peers {
  [id: string]: Peer.Instance
}

export interface PeerHandlerOptions {
  socket: ClientSocket
  peer: { id: string }
  dispatch: Dispatch
  getState: GetState
}

class PeerHandler {
  socket: ClientSocket
  peer: { id: string }
  dispatch: Dispatch
  getState: GetState
  decoder = new Decoder()
  textDecoder = new TextDecoder('utf-8')


  constructor (readonly options: PeerHandlerOptions) {
    this.socket = options.socket
    this.peer = options.peer
    this.dispatch = options.dispatch
    this.getState = options.getState
  }
  handleError = (err: Error) => {
    const { dispatch, getState, peer } = this
    debug('peer: %s, error %s', peer.id, err.stack)
    dispatch(NotifyActions.error('A peer connection error occurred'))
    const pc = getState().peers[peer.id]
    pc && pc.instance.destroy()
    dispatch(removePeer(peer.id))
  }
  handleSignal = (signal: SignalData) => {
    const { socket, peer } = this
    sdpDebug('local signal: %s, signal: %o', peer.id, signal)

    const payload = { peerId: peer.id, signal }
    socket.emit('signal', payload)
  }
  handleConnect = () => {
    const { dispatch, peer } = this
    debug('peer: %s, connect', peer.id)
    dispatch(NotifyActions.warning('Peer connection established'))


    dispatch(peerConnected({
      peerId: peer.id,
    }))
  }
  handleTrack = (
    track: MediaStreamTrack,
    stream: MediaStream,
    transceiver: RTCRtpTransceiver,
  ) => {
    const { peer, dispatch } = this
    const peerId = peer.id
    const streamId = stream.id
    const mid = transceiver.mid!

    debug('peer: %s, track: %s, stream: %s, mid: %s',
          peerId, track.id, stream.id, mid)

    // Listen to mute event to know when a track was removed
    // https://github.com/feross/simple-peer/issues/512
    track.onmute = () => {
      debug(
        'peer: %s, track mute (id: %s, stream.id: %s)',
        peerId, track.id, stream.id)
      dispatch(StreamActions.removeTrack({ peerId, track, streamId }))
    }

    function addTrack() {
      debug(
        'peer: %s, track unmute (id: %s, stream.id: %s)',
        peerId, track.id, stream.id)
      dispatch(StreamActions.addTrack({
        streamId,
        peerId,
        track,
        receiver: transceiver.receiver,
      }))
    }

    if (!track.muted) {
      addTrack()
    }
    track.onunmute = addTrack
  }
  handleData = (buffer: ArrayBuffer) => {
    const { dispatch, peer } = this

    const dataContainer = this.decoder.decode(buffer)
    if (!dataContainer) {
      // not all chunks have been received yet
      return
    }

    const { data } = dataContainer
    const message = JSON.parse(this.textDecoder.decode(data))

    debug('peer: %s, message: %o', peer.id, message)
    dispatch(addMessage(message))
  }
  handleClose = () => {
    const { dispatch, peer } = this
    dispatch(NotifyActions.error('Peer connection closed'))
    dispatch(removePeer(peer.id))
  }
}

export interface CreatePeerOptions {
  socket: ClientSocket
  peer: { id: string }
  initiator: boolean
  stream?: MediaStream
}

export function createPeer (options: CreatePeerOptions) {
  const { socket, peer, initiator, stream } = options

  return (dispatch: Dispatch, getState: GetState) => {
    const peerId = peer.id
    debug(
      'create peer: %s, hasStream: %s, initiator: %s',
      peerId, !!stream, initiator)
    dispatch(NotifyActions.warning('Connecting to peer...'))

    const oldPeer = getState().peers[peerId]
    if (oldPeer) {
      dispatch(NotifyActions.info('Cleaning up old connection...'))
      oldPeer.instance.destroy()
      dispatch(removePeer(peerId))
    }

    debug('Using ice servers: %o', peerConfig.iceServers)

    const pc = new Peer({
      initiator,
      config: {
        iceServers: peerConfig.iceServers,
        encodedInsertableStreams: peerConfig.encodedInsertableStreams,
        // legacy flags for insertable streams
        enableInsertableStreams: peerConfig.encodedInsertableStreams,
        forceEncodedVideoInsertableStreams:
          peerConfig.encodedInsertableStreams,
        forceEncodedAudioInsertableStreams:
          peerConfig.encodedInsertableStreams,
      },
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
      peer,
      dispatch,
      getState,
    })

    pc.once(constants.PEER_EVENT_ERROR, handler.handleError)
    pc.once(constants.PEER_EVENT_CONNECT, handler.handleConnect)
    pc.once(constants.PEER_EVENT_CLOSE, handler.handleClose)
    pc.on(constants.PEER_EVENT_SIGNAL, handler.handleSignal)
    pc.on(constants.PEER_EVENT_TRACK, handler.handleTrack)
    pc.on(constants.PEER_EVENT_DATA, handler.handleData)

    dispatch(addPeer({ peer: pc, peerId }))
  }
}

export interface AddPeerParams {
  peer: Peer.Instance
  peerId: string
}

export interface AddPeerAction {
  type: 'PEER_ADD'
  payload: AddPeerParams
}

export const addPeer = (payload: AddPeerParams): AddPeerAction => ({
  type: constants.PEER_ADD,
  payload,
})

export interface PeerConnectedParams {
  peerId: string
}

export interface PeerConnectedAction {
  type: 'PEER_CONNECTED'
  payload: PeerConnectedParams
}

export const peerConnected = (
  payload: PeerConnectedParams,
): PeerConnectedAction => ({
  type: constants.PEER_CONNECTED,
  payload,
})

export interface RemovePeerAction {
  type: 'PEER_REMOVE'
  payload: { peerId: string }
}

export interface RemoveAllPeersAction {
  type: 'PEER_REMOVE_ALL'
}

export const removePeer = (peerId: string): RemovePeerAction => ({
  type: constants.PEER_REMOVE,
  payload: { peerId },
})

export const removeAllPeers = (): RemoveAllPeersAction => ({
  type: constants.PEER_REMOVE_ALL,
})

export type PeerAction =
  AddPeerAction |
  RemovePeerAction |
  RemoveAllPeersAction |
  PeerConnectedAction
