const debug = require('debug')('peer-calls:video')

export function play () {
  let videos = window.document.querySelectorAll('video')
  Array.prototype.forEach.call(videos, (video, index) => {
    debug('playing video: %s', index)
    try {
      video.play()
    } catch (e) {
      debug('error playing video: %s', e.name)
    }
  })
}
