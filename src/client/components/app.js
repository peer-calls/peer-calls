const Alert = require('./alert.js')
const Input = require('./input.js')
const Notifications = require('./notifications.js')
const React = require('react')
const _ = require('underscore')
const activeStore = require('../store/activeStore.js')
const debug = require('debug')('peer-calls:app')
const dispatcher = require('../dispatcher/dispatcher.js')
const streamStore = require('../store/streamStore.js')

function app () {
  let streams = streamStore.getStreams()

  function play (event) {
    try {
      event.target.play()
    } catch (e) {
      debug('error starting video: %s', e.name)
    }
  }

  let videos = _.map(streams, (stream, userId) => {
    let url = stream.url

    function markActive (event) {
      play(event)
      dispatcher.dispatch({
        type: 'mark-active',
        userId: activeStore.isActive(userId) ? undefined : userId
      })
    }

    let className = 'video-container'
    className += activeStore.isActive(userId) ? ' active' : ''

    return (
      <div className={className} key={userId}>
        <video
          muted={userId === '_me_'}
          onClick={markActive}
          onLoadedMetadata={play}
          src={url}
        />
      </div>
    )
  })

  return (<div className='app'>
    <Alert />
    <Notifications />
    <Input />
    <div className='videos'>
      {videos}
    </div>
  </div>)
}

module.exports = app
