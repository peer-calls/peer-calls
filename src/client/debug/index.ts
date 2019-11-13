import { config, ICEServer } from '../../server/config'

async function checkTURNServer (turnConfig: ICEServer, timeoutDuration = 5000) {
  console.log('checking turn server', turnConfig)

  const timeout = new Promise<unknown>((resolve, reject) => {
    setTimeout(function () {
      reject(new Error('timed out'))
    }, timeoutDuration)
  })

  async function start() {
    const PeerConnection = window.RTCPeerConnection ||
      (
        window as unknown as { mozRTCPeerConnection: RTCPeerConnection }
      ).mozRTCPeerConnection||
      window.webkitRTCPeerConnection

    const pc = new PeerConnection({ iceServers: [turnConfig] })

    // create a bogus data channel
    pc.createDataChannel('')
    const sdp = await pc.createOffer()
    // sometimes sdp contains the ice candidates...
    if (sdp.sdp!.indexOf('typ relay') > -1) {
      return true
    }
    pc.setLocalDescription(sdp)

    return new Promise(resolve => {
      pc.onicecandidate = function (ice) {
        if (!ice ||
            !ice.candidate ||
            !ice.candidate.candidate ||
            !(ice.candidate.candidate.indexOf('typ relay') > -1)) {
          return
        }
        resolve(true)
      }
    })
  }

  return Promise.race([ timeout, start() ])
}

checkTURNServer(config.iceServers[0], 10000)
.then(console.log.bind(console))
.catch(console.error.bind(console))
