import SocketIOClient from 'socket.io-client'
import { baseUrl } from './window.js'
export default new SocketIOClient('', { path: baseUrl + '/ws' })
