'use strict';
if (window.localStorage && !window.localStorage.debug) {
  window.localStorage.debug = 'peer-calls:*';
}
window.__REACT_DEVTOOLS_GLOBAL_HOOK__ = {};

const App = require('./components/app.js');
const React = require('react');
const ReactDom = require('react-dom');
const activeStore = require('./store/activeStore.js');
const alertStore = require('./store/alertStore.js');
const debug = require('debug')('peer-calls:index');
const dispatcher = require('./dispatcher/dispatcher.js');
const getUserMedia = require('./browser/getUserMedia.js');
const handshake = require('./peer/handshake.js');
const notificationsStore = require('./store/notificationsStore.js');
const notify = require('./action/notify.js');
const socket = require('./socket.js');
const streamStore = require('./store/streamStore.js');

function play() {
  let videos = window.document.querySelectorAll('video');
  Array.prototype.forEach.call(videos, (video, index) => {
    debug('playing video: %s', index);
    try {
      video.play();
    } catch (e) {
      debug('error playing video: %s', e.name);
    }
  });
}

function render() {
  ReactDom.render(<App />, document.querySelector('#container'));
  play();
}

dispatcher.register(action => {
  if (action.type === 'play') play();
});

activeStore.addListener(render);
alertStore.addListener(render);
notificationsStore.addListener(render);
streamStore.addListener(render);

render();

const callId = window.document.getElementById('callId').value;

getUserMedia({ video: true, audio: false })
.then(stream => {
  dispatcher.dispatch({
    type: 'add-stream',
    userId: '_me_',
    stream
  });
})
.catch(() => {
  notify.alert('Could not get access to microphone & camera');
});

socket.once('connect', () => {
  notify.warn('Connected to server socket');
  debug('socket connected');
  getUserMedia({ video: true, audio: true })
  .then(stream => {
    debug('forwarding stream to handshake');
    handshake.init(socket, callId, stream);
  })
  .catch(err => {
    notify.alert('Could not get access to camera!', true);
    debug('error getting media: %s %s', err.name, err.message);
  });
});

socket.on('disconnect', () => {
  notify.error('Server socket disconnected');
});
