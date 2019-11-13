import SocketIOClient from 'socket.io-client'
import { baseUrl } from './window.js'
export default SocketIOClient('', { path: baseUrl + '/ws' })
