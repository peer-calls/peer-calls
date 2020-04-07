import { GetAsyncAction, makeAction } from '../async'
import { DIAL, HANG_UP, SOCKET_EVENT_USERS } from '../constants'
import socket from '../socket'
import store, { ThunkResult } from '../store'
import { callId, userId } from '../window'
import * as NotifyActions from './NotifyActions'
import * as SocketActions from './SocketActions'

export interface InitAction {
  type: 'INIT'
  payload: Promise<void>
}

interface InitializeAction {
  type: 'INIT'
}

const initialize = (): InitializeAction => ({
  type: 'INIT',
})

export const init = (): ThunkResult<Promise<void>> => async dispatch => {
  return new Promise(resolve => {
    socket.on('connect', () => {
      dispatch(NotifyActions.warning('Connected to server socket'))
      dispatch(initialize())
      resolve()
    })
    socket.on('disconnect', () => {
      dispatch(NotifyActions.error('Server socket disconnected'))
    })
  })
}

export const dial = makeAction(
  DIAL,
  () => new Promise<void>((resolve, reject) => {
    SocketActions.handshake({
      socket,
      roomName: callId,
      userId,
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
  SocketActions.removeEventListeners(socket)
  return {
    type: HANG_UP,
  }
}

export type DialAction = GetAsyncAction<ReturnType<typeof dial>>
