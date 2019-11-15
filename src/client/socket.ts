import SocketIOClient from 'socket.io-client'
import { baseUrl } from './window'
export default SocketIOClient('', { path: baseUrl + '/ws' })
