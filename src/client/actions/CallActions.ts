import socket from '../socket'
import { Dispatch, ThunkResult } from '../store'
import { callId } from '../window'
import { ClientSocket } from '../socket'
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
  const socket = await dispatch(connect())

  dispatch(SocketActions.handshake({
    socket,
    roomName: callId,
  }))

  dispatch(initialize())
}

export const connect = () => (dispatch: Dispatch) => {
  return new Promise<ClientSocket>(resolve => {
    socket.once('connect', () => {
      resolve(socket)
    })
    socket.on('connect', () => {
      dispatch(NotifyActions.warning('Connected to server socket'))
    })
    socket.on('disconnect', () => {
      dispatch(NotifyActions.error('Server socket disconnected'))
    })
  })
}
