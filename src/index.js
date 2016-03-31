#!/usr/bin/env node
'use strict';
if (!process.env.DEBUG) {
  process.env.DEBUG = 'peer-calls:*';
}

const express = require('express');
const app = express();
const http = require('http').Server(app);
const io = require('socket.io')(http);
const path = require('path');
const os = require('os');

const handleSocket = require('./server/socket.js');

app.set('view engine', 'jade');
app.set('views', path.join(__dirname, 'views'));

app.use('/res', express.static(path.join(__dirname, 'res')));

if (path.basename(__dirname) === 'dist') {
  app.use('/js', express.static(path.join(__dirname, 'js')));
  app.use('/less', express.static(path.join(__dirname, 'less')));
} else {
  const browserify = require('browserify-middleware');
  const less = require('less-middleware');
  browserify.settings({
    transform: ['babelify']
  });

  const tempDir = path.join(os.tmpDir(), 'node-mpv-css-cache');
  app.use('/js', browserify(path.join(__dirname, './js')));
  app.use('/less', less(path.join(__dirname, './less'), { dest: tempDir}));
  app.use('/less', express.static(tempDir));
  app.use('/less/fonts', express.static(
    path.join(__dirname, './less/fonts')));
}

app.get('/', (req, res) => res.render('index'));

io.on('connection', socket => handleSocket(socket, io));

let port = process.env.PORT || 3000;
let ifaces = os.networkInterfaces();
http.listen(port, function() {
  Object.keys(ifaces).forEach(ifname =>
    ifaces[ifname].forEach(iface =>
      console.log('listening on', iface.address, 'and port', port)));
});
