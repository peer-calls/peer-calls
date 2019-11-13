jest.mock('../window.js')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import Video from './Video.js'
import { MediaStream } from '../window.js'

describe('components/Video', () => {

  class VideoWrapper extends React.PureComponent {
    static propTypes = Video.propTypes
    constructor () {
      super()
      this.state = {}
    }
    render () {
      return <Video
        videos={this.props.videos}
        active={this.props.active}
        stream={this.state.stream || this.props.stream}
        onClick={this.props.onClick}
        userId="test"
        muted={this.props.muted}
        mirrored={this.props.mirrored}
      />
    }
  }

  let component, videos, video, onClick, mediaStream, url, wrapper
  function render (flags = {}) {
    videos = {}
    onClick = jest.fn()
    mediaStream = new MediaStream()
    component = TestUtils.renderIntoDocument(
      <VideoWrapper
        videos={videos}
        active={flags.active || false}
        stream={{ mediaStream, url }}
        onClick={onClick}
        userId="test"
        muted={flags.muted || false}
        mirrored={flags.mirrored}
      />
    )
    wrapper = ReactDOM.findDOMNode(component)
    video = TestUtils.findRenderedComponentWithType(component, Video)
  }

  describe('render', () => {
    it('should not fail', () => {
      render({})
    })

    it('Mirrored and active propogate to rendered classes', () => {
      render({ active: true, mirrored: true })
      expect(wrapper.className).toBe('video-container active mirrored')
    })
  })

  describe('componentDidUpdate', () => {
    describe('src', () => {
      beforeEach(() => {
        render()
        delete video.refs.video.srcObject
      })
      it('updates src only when changed', () => {
        mediaStream = new MediaStream()
        component.setState({
          stream: { url: 'test', mediaStream }
        })
        expect(video.refs.video.src).toBe('http://localhost/test')
        component.setState({
          stream: { url: 'test', mediaStream }
        })
      })
      it('updates srcObject only when changed', () => {
        video.refs.video.srcObject = null
        mediaStream = new MediaStream()
        component.setState({
          stream: { url: 'test', mediaStream }
        })
        expect(video.refs.video.srcObject).toBe(mediaStream)
        component.setState({
          stream: { url: 'test', mediaStream }
        })
      })
    })
  })

})
