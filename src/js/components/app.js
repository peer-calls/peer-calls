const React = require('react');
const _ = require('underscore');
const activeStore = require('../store/activeStore.js');
const createObjectURL = require('../browser/createObjectURL');
const dispatcher = require('../dispatcher/dispatcher.js');
const streamStore = require('../store/streamStore.js');

function app() {
  let streams = streamStore.getStreams();

  function play(event) {
    event.target.play();
  }

  let videos = _.map(streams, (stream, userId) => {
    let url = createObjectURL(stream);

    function markActive() {
      if (activeStore.isActive(userId)) return;
      dispatcher.dispatch({
        type: 'mark-active',
        userId
      });
    }

    let className = 'video-container';
    className += activeStore.isActive(userId) ? ' active' : '';

    return (
      <div className={className} key={userId}>
        <video onClick={markActive} onLoadedMetadata={play} src={url} />
      </div>
    );
  });

  return (<div className="app">
    <div className="videos">
      {videos}
    </div>
  </div>);
}

module.exports = app;
