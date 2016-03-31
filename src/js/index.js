if (window.localStorage && !window.localStorage.debug) {
  window.localStorage.debug = 'video-client:*';
}

const React = require('react');
const ReactDom = require('react-dom');

const App = require('./components/app.js');
const handshake = require('./peer/handshake.js');
const debug = require('debug')('video-client:index');
const getUserMedia = require('./browser/getUserMedia.js');
const socket = require('./socket.js');
const activeStore = require('./store/activeStore.js');
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
