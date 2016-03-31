'use strict';
const EventEmitter = require('events');
const debug = require('debug')('peer-calls:streamStore');
const dispatcher = require('../dispatcher/dispatcher.js');

const emitter = new EventEmitter();
const addListener = cb => emitter.on('change', cb);
const removeListener = cb => emitter.removeListener('change', cb);

const streams = {};

const handlers = {
  'add-stream': ({ userId, stream }) => {
    debug('add-stream, user: %s', userId);
    streams[userId] = stream;
  },
  'remove-stream': ({ userId }) => {
    debug('remove-stream, user: %s', userId);
    delete streams[userId];
  }
};

const dispatcherIndex = dispatcher.register(action => {
  let handle = handlers[action.type];
  if (!handle) return;
  handle(action);
  emitter.emit('change');
});

function getStream(userId) {
  return streams[userId];
}

function getStreams() {
  return streams;
}

module.exports = {
  dispatcherIndex,
  addListener,
  removeListener,
  getStream,
  getStreams
};
