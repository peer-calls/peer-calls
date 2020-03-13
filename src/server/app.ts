import { config } from './config'
import _debug from 'debug'
import bodyParser from 'body-parser'
import express from 'express'
import handleSocket from './socket'
import path from 'path'
import { createServer } from './server'
import SocketIO from 'socket.io'
import call from './routes/call'
import index from './routes/index'
import ejs from 'ejs'
import { MemoryStore } from './store'

const debug = _debug('peercalls')
const logRequest = _debug('peercalls:requests')

const BASE_URL: string = config.baseUrl
const SOCKET_URL = `${BASE_URL}/ws`

debug(`WebSocket URL: ${SOCKET_URL}`)

const app = express()
const server = createServer(config, app)
export const io = SocketIO(server, { path: SOCKET_URL })

app.set('x-powered-by', false)
app.locals.version = require('../../package.json').version
app.locals.baseUrl = BASE_URL
// eslint-disable-next-line
app.engine('html', ejs.renderFile as any)

app.set('view engine', 'html')
app.set('views', path.join(__dirname, '../../views'))

app.use((req, res, next) => {
  const start = Date.now()
  res.on('finish', () => {
    const duration = Date.now() - start
    logRequest('%s %s %sms', req.method, req.originalUrl, duration)
  })
  next()
})
app.use(bodyParser.urlencoded({ extended: false }))

const router = express.Router()
router.use('/res', express.static(path.join(__dirname, '../../res')))
router.use('/static', express.static(path.join(__dirname, '../../build')))
router.use('/call', call)
router.use('/', index)
app.use(BASE_URL, router)

const store = new MemoryStore()
io.on('connection', socket => handleSocket(socket, io, store))

export default server
