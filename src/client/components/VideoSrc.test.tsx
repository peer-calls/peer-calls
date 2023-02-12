jest.mock('../window')

import React, { ReactEventHandler } from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { Dim } from '../frame'
import { MediaStream } from '../window'
import VideoSrc, { VideoSrcProps } from './VideoSrc'

describe('components/VideoSrc', () => {
  interface State {
    src: string | undefined
    srcObject: MediaStream | null
  }

  class VideoSrcWrapper extends React.PureComponent<VideoSrcProps, State> {
    ref = React.createRef<VideoSrc>()

    state = {
      src: undefined,
      srcObject: null,
    }

    render() {
      return <VideoSrc
        {...this.props}
        ref={this.ref}
        src={this.state.src || this.props.src}
        srcObject={this.state.srcObject || this.props.srcObject}
      />
    }
  }

  let component: VideoSrcWrapper
  let videoSrc: VideoSrc
  let onLoadedMetadata: jest.MockedFunction<ReactEventHandler<HTMLVideoElement>>
  let onClick: jest.MockedFunction<ReactEventHandler<HTMLVideoElement>>
  let onResize: jest.MockedFunction<(dimensions: Dim) => void>
  let src: string | undefined
  let srcObject: MediaStream | null = null

  async function render () {
    const div = document.createElement('div')
    component = await new Promise<VideoSrcWrapper>(resolve => {
      ReactDOM.render(
        <VideoSrcWrapper
          ref={instance => resolve(instance!)}
          id="test"
          autoPlay
          mirrored
          muted
          onClick={onClick}
          onLoadedMetadata={onLoadedMetadata}
          onResize={onResize}
          src={src}
          srcObject={srcObject}
          objectFit='contain'
        />,
        div,
      )
    })
    videoSrc = TestUtils.findRenderedComponentWithType(component, VideoSrc)
    // wrapper = div.children[0]
  }

  beforeEach(() => {
    src = undefined
    srcObject = null
  })

  describe('render', () => {
    it('renders and sets src accordingly', async () => {
      src = 'http://localhost/test.mp4'
      await render()
      expect(videoSrc.videoRef.current!.src).toBe(src)
    })
  })

  describe('componentDidUpdate', () => {
    it('updates srcObject', async () => {
      await render()
      const stream = new MediaStream()
      videoSrc.videoRef.current!.srcObject = null
      component.setState({
        srcObject: stream,
      })
      expect(videoSrc.videoRef.current!.srcObject).toBe(stream)
      expect(videoSrc.videoRef.current!.muted).toBe(true)
    })
  })
})
