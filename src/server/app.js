#!/usr/bin/env node
'use strict';
const express = require('express');
const handleSocket = require('./socket.js');
const os = require('os');
const path = require('path');
const uuid = require('uuid');

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
  app.use('/less', less(path.join(__dirname, '../less'), { dest: tempDir}));
  app.use('/css', express.static(tempDir));
  app.use('/css/fonts', express.static(
    path.join(__dirname, '../less/fonts')));
}

app.get('/', (req, res) => res.render('index'));
app.get('/call/', (req, res) => {
  let prefix = 'call/';
  if (req.url.charAt(req.url.length - 1) === '/') prefix = '';
  res.redirect(prefix + uuid.v4());
});
app.get('/call/:callId', (req, res) => {
  res.render('call', {
    callId: encodeURIComponent(req.params.callId)
  });
});

io.on('connection', socket => handleSocket(socket, io));

module.exports = { http, app };
