import { makeAction, GetAsyncAction } from '../async'
import { DIAL, SOCKET_EVENT_READY, SOCKET_EVENT_USERS } from '../constants'
import socket from '../socket'
import { ThunkResult } from '../store'
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

export const init = (): ThunkResult<Promise<void>> =>
async (dispatch, getState) => {
  return new Promise(resolve => {
    socket.on('connect', () => {
      dispatch(NotifyActions.warning('Connected to server socket'))
      dispatch(SocketActions.handshake({
        socket,
        roomName: callId,
        userId,
      }))
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
    socket.emit(SOCKET_EVENT_READY, {
      room: callId,
      userId,
    })
    socket.once(SOCKET_EVENT_USERS, () => resolve())
    setTimeout(reject, 10000, new Error('Dial timed out!'))
  }),
)

export type DialAction = GetAsyncAction<ReturnType<typeof dial>>
