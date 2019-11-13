import * as NotifyActions from './NotifyActions'
import * as SocketActions from './SocketActions'
import * as StreamActions from './StreamActions'
import * as constants from '../constants'
import socket from '../socket'
import { callId, getUserMedia } from '../window'
import { Dispatch } from 'redux'
import { GetState } from '../store'

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

export const init = async (dispatch: Dispatch, getState: GetState) => {
  dispatch(initialize())

  const socket = await connect(dispatch)
  const stream = await getCameraStream(dispatch)

  SocketActions.handshake({
    socket,
    roomName: callId,
    stream,
  })(dispatch, getState)
}

export async function connect (dispatch: Dispatch) {
  return new Promise<SocketIOClient.Socket>(resolve => {
    socket.once('connect', () => {
      resolve(socket)
    })
    socket.on('connect', () => {
      NotifyActions.warning('Connected to server socket')(dispatch)
    })
    socket.on('disconnect', () => {
      NotifyActions.error('Server socket disconnected')(dispatch)
    })
  })
}

export async function getCameraStream (dispatch: Dispatch) {
  try {
    const stream = await getUserMedia({
      video: { facingMode: 'user' },
      audio: true,
    })
    dispatch(StreamActions.addStream({ stream, userId: constants.ME }))
    return stream
  } catch (err) {
    dispatch(NotifyActions.alert('Could not get access to microphone & camera'))
    return
  }
}
