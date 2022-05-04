import Peer from 'simple-peer'
import socket from './socket'
import { config } from './window'

const { peerConfig, peerId, callId } = config

const { iceServers, encodedInsertableStreams } = peerConfig

const $container = document.getElementById('container')!
socket.reconnectTimeout = 0

socket.on('connect', () => {
  socket.on('users', ({initiator, peerIds}) => {
    console.log('users', peerIds)

    peerIds.forEach(remotePeerId => {
      if (peerId === remotePeerId) {
        console.log('skipping current user', peerIds)

        return
      }

      const peer = new Peer({
        initiator: initiator === remotePeerId,
        config: { iceServers, encodedInsertableStreams },
        trickle: false,
        // Allow the peer to receive video, even if it's not sending stream:
        // https://github.com/feross/simple-peer/issues/95
        //offerConstraints: {
        //  offerToReceiveAudio: true,
        //  offerToReceiveVideo: true,
        //},
      })

      peer.on('signal', signal => {
        console.log('signal', signal)

        socket.emit('signal', {
          peerId: remotePeerId,
          signal,
        })
      })

      socket.on('signal', payload => {
        peer.signal(payload.signal)
      })


      // ;(peer as any).addTransceiver('video', {
      //   direction:'sendonly',
      // })

      peer.on('connect', () => {
        console.log('peer connect')
        navigator.mediaDevices.getUserMedia({
          video: true,
          audio: false,
        })
        .then(stream => {
          const el = document.createElement('video')
          el.style.width = '200px'
          el.style.backgroundColor = '#555'
          el.srcObject = stream
          el.autoplay = true
          el.controls = true
          $container.appendChild(el)

          stream.getTracks().forEach(track => {
            console.log(
              'local track', track.id, track.label,
              'muted?', track.muted, 'enabled?', track.enabled)
            peer.addTrack(track, stream)
          })

        })
        .catch(err => {
          console.error(err)
        })
      })

      peer.on('track', (track: MediaStreamTrack, stream: MediaStream) => {
        console.log(
          'remote track', track.id, track.label,
          'muted?', track.muted, 'enabled?', track.enabled)
        track.enabled = true
        const el = document.createElement(track.kind) as HTMLVideoElement
        el.style.width = '200px'
        el.style.backgroundColor = '#555'
        el.srcObject = stream
        el.autoplay = true
        el.controls = true
        $container.appendChild(el)

        track.onunmute = function(event) {
          console.log('track unmuted', track.id)
        }

        track.onmute = function(event) {
          console.log('track muted', track.id)
          $container.removeChild(el)
        }
      })
    })
  })

  console.log('emitting ready')

  socket.emit('ready', {
    room: callId,
    nickname: 'test',
    peerId,
  })

})
