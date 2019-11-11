import * as NotifyActions from './NotifyActions.js'
import * as SocketActions from './SocketActions.js'
import * as StreamActions from './StreamActions.js'
import * as constants from '../constants.js'
import Promise from 'bluebird'
import socket from '../socket.js'
import { callId, getUserMedia } from '../window.js'

export const init = () => dispatch => {
  return dispatch({
    type: constants.INIT,
    payload: Promise.all([
      connect()(dispatch),
      getCameraStream()(dispatch)
    ])
    .spread((socket, stream) => {
      dispatch(SocketActions.handshake({
        socket,
        roomName: callId,
        stream
      }))
    })
  })
}

export const connect = () => dispatch => {
  return new Promise(resolve => {
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

export const getCameraStream = () => dispatch => {
  return getUserMedia({ video: { facingMode: 'user' }, audio: true })
  .then(stream => {
    dispatch(StreamActions.addStream({ stream, userId: constants.ME }))
    return stream
  })
  .catch(() => {
    dispatch(NotifyActions.alert('Could not get access to microphone & camera'))
    return null
  })
}
