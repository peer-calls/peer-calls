'use strict';
window.__REACT_DEVTOOLS_GLOBAL_HOOK__ = {};

const App = require('./components/app.js');
const React = require('react');
const ReactDom = require('react-dom');
const activeStore = require('./store/activeStore.js');
const alertStore = require('./store/alertStore.js');
const call = require('./call.js');
const debug = require('debug')('peer-calls:index');
const notificationsStore = require('./store/notificationsStore.js');
const play = require('./browser/video.js').play;
const streamStore = require('./store/streamStore.js');

function render() {
  debug('rendering');
  ReactDom.render(<App />, document.querySelector('#container'));
  play();
}

activeStore.addListener(render);
alertStore.addListener(render);
notificationsStore.addListener(render);
streamStore.addListener(render);

render();

call.init();
