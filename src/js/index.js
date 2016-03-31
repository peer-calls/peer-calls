'use strict';
if (window.localStorage && !window.localStorage.debug) {
  window.localStorage.debug = 'peercalls:*';
}

const App = require('./components/app.js');
const React = require('react');
const ReactDom = require('react-dom');
const activeStore = require('./store/activeStore.js');
const debug = require('debug')('peercalls:index');
const dispatcher = require('./dispatcher/dispatcher.js');
const getUserMedia = require('./browser/getUserMedia.js');
const handshake = require('./peer/handshake.js');
const socket = require('./socket.js');
const streamStore = require('./store/streamStore.js');

function render() {
  ReactDom.render(<App />, document.querySelector('#container'));
}

streamStore.addListener(() => () => {
  debug('streamStore - change');
  debug(streamStore.getStreams());
});
streamStore.addListener(render);
activeStore.addListener(render);

render();

getUserMedia({ video: true, audio: false })
.then(stream => {
  dispatcher.dispatch({
    type: 'add-stream',
    userId: '_me_',
    stream
  });
});

socket.once('connect', () => {
  debug('socket connected');
  getUserMedia({ video: true, audio: true })
  .then(stream => {
    debug('forwarding stream to handshake');
    handshake.init(socket, 'test', stream);
  })
  .catch(err => {
    debug('error getting media: %s %s', err.name, err.message);
  });
});
