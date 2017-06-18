const iceServers = require('./iceServers.js')

function noop () {}

function checkTURNServer (turnConfig, timeout) {
  console.log('checking turn server', turnConfig)

  return new Promise(function (resolve, reject) {
    setTimeout(function () {
      if (promiseResolved) return
      resolve(false)
      promiseResolved = true
    }, timeout || 5000)

    let promiseResolved = false
    const PeerConnection = window.RTCPeerConnection ||
      window.mozRTCPeerConnection ||
      window.webkitRTCPeerConnection

    const pc = new PeerConnection({ iceServers: [turnConfig] })

    // create a bogus data channel
    pc.createDataChannel('')
    pc.createOffer(function (sdp) {
      // sometimes sdp contains the ice candidates...
      if (sdp.sdp.indexOf('typ relay') > -1) {
        promiseResolved = true
        resolve(true)
      }
      pc.setLocalDescription(sdp, noop, noop)
    }, noop)

    pc.onicecandidate = function (ice) {
      if (promiseResolved ||
          !ice ||
          !ice.candidate ||
          !ice.candidate.candidate ||
          !(ice.candidate.candidate.indexOf('typ relay') > -1)) {
        return
      }
      promiseResolved = true
      resolve(true)
    }
  })
}

checkTURNServer(iceServers[0], 10000)
.then(console.log.bind(console))
.catch(console.error.bind(console))
