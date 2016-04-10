#!/usr/bin/env node
'use strict';
const express = require('express');
const handleSocket = require('./socket.js');
const os = require('os');
const path = require('path');

//Require in express.Router Middleware. 
const callRouter = require('./routes/call');
const siteRouter = require('./routes/index');

const app = express();
const http = require('http').Server(app);
const io = require('socket.io')(http);

app.set('view engine', 'jade');
app.set('views', path.join(__dirname, '../views'));

app.use('/res', express.static(path.join(__dirname, '../res')));

if (__dirname.indexOf('/dist/') >= 0 || __dirname.indexOf('\\dist\\') >= 0) {
  app.use('/js', express.static(path.join(__dirname, '../client')));
  app.use('/css', express.static(path.join(__dirname, '../css')));
} else {
  const browserify = require('browserify-middleware');
  const less = require('less-middleware');
  browserify.settings({
    transform: ['babelify']
  });

  const tempDir = path.join(os.tmpDir(), 'node-peer-calls-cache');
  app.use('/js', browserify(path.join(__dirname, '../client')));
  app.use('/css', less(path.join(__dirname, '../less'), { dest: tempDir}));
  app.use('/css', express.static(tempDir));
  app.use('/css/fonts', express.static(
    path.join(__dirname, '../less/fonts')));
}

//using Express.Router Middleware
app.use('/call', callRouter);
app.use('/', siteRouter);



io.on('connection', socket => handleSocket(socket, io));

module.exports = { http, app };
