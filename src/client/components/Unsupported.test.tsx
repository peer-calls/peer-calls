import React from 'react'
import ReactDOM from 'react-dom'
import { Unsupported } from './Unsupported'

describe('components/Unsupported', () => {

  let node: Element
  async function render () {
    const div = document.createElement('div')
    await new Promise<Unsupported>(resolve => {
      ReactDOM.render(
        <Unsupported
          ref={ref => resolve(ref!)}
        />,
        div,
      )
    })
    node = div.children[0]
  }

  describe('render', () => {
    describe('required browser features missing', () => {
      it('shows a warning', async () => {
        await render()
        expect(node).toBeDefined()
        expect(node.textContent).toMatch(/unsupported browser/i)
      })
    })

    describe('required browser features present', () => {
      const g = global as any
      beforeEach(() => {
        g.navigator.mediaDevices = {}
        g.navigator.mediaDevices.getUserMedia = () => undefined
        g.navigator.mediaDevices.enumerateDevices = () => undefined
        g.TextEncoder = () => undefined
        g.TextDecoder = () => undefined
        g.MediaStream = () => undefined
        g.MediaStreamTrack = () => undefined
        g.Worker = () => undefined
        g.RTCPeerConnection = function () {}
        g.RTCPeerConnection.prototype.createDataChannel = () => undefined
      })
      afterEach(() => {
        delete g.navigator.mediaDevices
        delete g.TextEncoder
        delete g.TextDecoder
        delete g.MediaStream
        delete g.MediaStreamTrack
        delete g.Worker
        delete g.RTCPeerConnection
      })
      it('renders nothing', async () => {
        await render()
        expect(node).toBe(undefined)
      })
    })
  })

})
