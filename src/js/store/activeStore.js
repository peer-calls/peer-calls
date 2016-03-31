'use strict';
const EventEmitter = require('events');
const debug = require('debug')('video-client:activeStore');
const dispatcher = require('../dispatcher/dispatcher.js');

const emitter = new EventEmitter();
const addListener = cb => emitter.on('change', cb);
const removeListener = cb => emitter.removeListener('change', cb);

let active;

const handlers = {
  'add-stream': ({ userId }) => {
    active = userId;
  },
  'mark-active': ({ userId }) => {
    debug('mark-active, userId: %s', userId);
    active = userId;
  }
};

const dispatcherIndex = dispatcher.register(action => {
  let handle = handlers[action.type];
  if (!handle) return;
  handle(action);
  emitter.emit('change');
});

function getActive() {
  return active;
}

function isActive(test) {
  return active === test;
}

module.exports = {
  dispatcherIndex,
  addListener,
  removeListener,
  getActive,
  isActive
};
