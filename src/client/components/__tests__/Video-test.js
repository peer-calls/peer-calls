jest.mock('../../window.js')
import React from 'react'
import TestUtils from 'react-dom/test-utils'
import Video from '../Video.js'
import { MediaStream } from '../../window.js'

describe('components/Video', () => {

  class VideoWrapper extends React.PureComponent {
    static propTypes = Video.propTypes
    constructor () {
      super()
      this.state = {}
    }
    render () {
      return <Video
        active={this.props.active}
        stream={this.state.stream || this.props.stream}
        onClick={this.props.onClick}
        userId="test"
      />
    }
  }

  let component, video, onClick, mediaStream, url
  function render () {
    onClick = jest.fn()
    mediaStream = new MediaStream()
    component = TestUtils.renderIntoDocument(
      <VideoWrapper
        active
        stream={{ mediaStream, url }}
        onClick={onClick}
        userId="test"
      />
    )
    video = TestUtils.findRenderedComponentWithType(component, Video)
  }

  describe('render', () => {
    it('should not fail', () => {
      render()
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
