'use strict';
if (window.localStorage && !window.localStorage.debug) {
  window.localStorage.debug = 'peer-calls:*';
}

const App = require('./components/app.js');
const React = require('react');
const ReactDom = require('react-dom');
const activeStore = require('./store/activeStore.js');
const debug = require('debug')('peer-calls:index');
const dispatcher = require('./dispatcher/dispatcher.js');
const getUserMedia = require('./browser/getUserMedia.js');
const handshake = require('./peer/handshake.js');
const socket = require('./socket.js');
const streamStore = require('./store/streamStore.js');

function play() {
  let videos = window.document.querySelectorAll('video');
  Array.prototype.forEach.call(videos, (video, index) => {
    debug('playing video: %s', index);
    video.play();
  });
}

function render() {
  ReactDom.render(<App />, document.querySelector('#container'));
  play();
}

dispatcher.register(action => {
  if (action.type === 'play') play();
});

streamStore.addListener(() => () => {
  debug('streamStore - change');
  debug(streamStore.getStreams());
});
streamStore.addListener(render);
activeStore.addListener(render);

render();

const callId = window.document.getElementById('callId').value;

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
    handshake.init(socket, callId, stream);
  })
  .catch(err => {
    debug('error getting media: %s %s', err.name, err.message);
  });
});
