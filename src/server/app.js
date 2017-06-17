#!/usr/bin/env node
'use strict'
const express = require('express')
const handleSocket = require('./socket.js')
const path = require('path')

const app = express()
const http = require('http').Server(app)
const io = require('socket.io')(http)

app.set('view engine', 'pug')
app.set('views', path.join(__dirname, '../views'))

app.use('/res', express.static(path.join(__dirname, '../res')))
app.use('/static', express.static(path.join(__dirname, '../../build')))
app.use('/call', require('./routes/call.js'))
app.use('/', require('./routes/index.js'))

io.on('connection', socket => handleSocket(socket, io))

module.exports = { http, app }
