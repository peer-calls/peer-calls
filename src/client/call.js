const debug = require('debug')('peer-calls:call');
const dispatcher = require('./dispatcher/dispatcher.js');
const getUserMedia = require('./browser/getUserMedia.js');
const play = require('./browser/video.js').play;
const notify = require('./action/notify.js');
const handshake = require('./peer/handshake.js');
const socket = require('./socket.js');

dispatcher.register(action => {
  if (action.type === 'play') play();
});

function init() {
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

}

module.exports = { init };
