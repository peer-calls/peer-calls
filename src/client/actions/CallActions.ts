import { GetAsyncAction, makeAction } from '../async'
import { DIAL, HANG_UP, ME, SOCKET_CONNECTED, SOCKET_DISCONNECTED, SOCKET_EVENT_HANG_UP, SOCKET_EVENT_USERS } from '../constants'
import socket from '../socket'
import store, { ThunkResult } from '../store'
import { config } from '../window'
import * as NotifyActions from './NotifyActions'
import { removeAllPeers } from './PeerActions'
import * as SocketActions from './SocketActions'

const { callId, peerId } = config

export interface ConnectedAction {
  type: 'SOCKET_CONNECTED'
}

const connected = (): ConnectedAction => ({
  type: SOCKET_CONNECTED,
})

export interface DisconnectedAction {
  type: 'SOCKET_DISCONNECTED'
}

const disconnected = (): DisconnectedAction => ({
  type: SOCKET_DISCONNECTED,
})

export const init = (): ThunkResult<Promise<void>> => async (
  dispatch, getState,
) => {
  return new Promise(resolve => {
    socket.on('connect', () => {
      dispatch(NotifyActions.warning('Connected to server socket'))
      dispatch(connected())

      const state = getState()
      const nickname = state.nicknames[ME]

      // Redial if the previous state was in-call, for example if the server
      // was restarted and websocket connection lost.
      if (state.media.dialState === 'in-call') {
        // Destroy all peers so we can start anew. It usually takes some time
        // for the peer connections to realize it has been disconnected so we
        // destroy all peers first so we can have a clean state. But we don't
        // want to call hangUp because that would remove the a/v streams.
        dispatch(removeAllPeers())

        dispatch(NotifyActions.info('Reconnecting to peer(s)...'))

        // restarted mid-call.
        dispatch(
          dial({
            nickname,
          }),
        )
        .catch(() => {
          dispatch(NotifyActions.error('Dial timed out.'))
        })
      }

      resolve()
    })
    socket.on('disconnect', () => {
      dispatch(NotifyActions.error('Server socket disconnected'))
      dispatch(disconnected())
    })
  })
}

export interface DialParams {
  nickname: string
}

export const dial = makeAction(
  DIAL,
  (params: DialParams) => new Promise<void>((resolve, reject) => {
    window.onbeforeunload = () =>
      'This will abort the current call - are you sure you wish to proceed?'
    SocketActions.handshake({
      nickname: params.nickname,
      socket,
      roomName: callId,
      peerId,
      store,
    })
    socket.once(SOCKET_EVENT_USERS, () => resolve())
    setTimeout(reject, 10000, new Error('Dial timed out!'))
  }),
)

export type HangUpAction = {
  type: 'HANG_UP'
}

export const hangUp = (): HangUpAction => {
  socket.emit(SOCKET_EVENT_HANG_UP, { peerId })
  SocketActions.removeEventListeners(socket)
  window.onbeforeunload = null
  return {
    type: HANG_UP,
  }
}

export type DialAction = GetAsyncAction<ReturnType<typeof dial>>
