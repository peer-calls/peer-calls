#!/usr/bin/env node
'use strict'
const config = require('config')
const debug = require('debug')('peercalls')
const express = require('express')
const handleSocket = require('./socket.js')
const path = require('path')
const { createServer } = require('./server.js')

const BASE_URL = config.get('baseUrl')
const SOCKET_URL = `${BASE_URL}/ws`

debug(`WebSocket URL: ${SOCKET_URL}`)

const app = express()
const server = createServer(config, app)
const io = require('socket.io')(server, { path: SOCKET_URL })

app.locals.version = require('../../package.json').version
app.locals.baseUrl = BASE_URL

app.set('view engine', 'pug')
app.set('views', path.join(__dirname, '../views'))
app.use(express.static(path.join(__dirname, '../../build')))

const router = express.Router()
router.use('/call', require('./routes/call.js'))
router.use('/', require('./routes/index.js'))
app.use(BASE_URL, router)

io.on('connection', socket => handleSocket(socket, io))

module.exports = server
