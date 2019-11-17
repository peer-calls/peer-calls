jest.mock('../window')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { AddStreamPayload } from '../actions/StreamActions'
import Video, { VideoProps } from './Video'
import { MediaStream } from '../window'

describe('components/Video', () => {

  interface VideoState {
    stream: null | AddStreamPayload
  }

  const play = jest.fn()

  class VideoWrapper extends React.PureComponent<VideoProps, VideoState> {
    ref = React.createRef<Video>()

    state = {
      stream: null,
    }

    render () {
      return <Video
        ref={this.ref}
        videos={this.props.videos}
        active={this.props.active}
        stream={this.state.stream || this.props.stream}
        onClick={this.props.onClick}
        play={this.props.play}
        userId="test"
        muted={this.props.muted}
        mirrored={this.props.mirrored}
      />
    }
  }

  let component: VideoWrapper
  let videos: Record<string, unknown> = {}
  let video: Video
  let onClick: (userId: string) => void
  let mediaStream: MediaStream
  let url: string
  let wrapper: Element

  interface Flags {
    active: boolean
    muted: boolean
    mirrored: boolean
  }
  const defaultFlags: Flags = {
    active: false,
    muted: false,
    mirrored: false,
  }
  async function render (args?: Partial<Flags>) {
    const flags: Flags = Object.assign({}, defaultFlags, args)
    videos = {}
    onClick = jest.fn()
    mediaStream = new MediaStream()
    const div = document.createElement('div')
    component = await new Promise<VideoWrapper>(resolve => {
      ReactDOM.render(
        <VideoWrapper
          ref={instance => resolve(instance!)}
          videos={videos}
          active={flags.active}
          stream={{ stream: mediaStream, url, userId: 'test' }}
          onClick={onClick}
          play={play}
          userId="test"
          muted={flags.muted}
          mirrored={flags.mirrored}
        />,
        div,
      )
    })
    video = TestUtils.findRenderedComponentWithType(component, Video)
    wrapper = div.children[0]
  }

  describe('render', () => {
    it('should not fail', async () => {
      await render()
    })

    it('Mirrored and active propogate to rendered classes', async () => {
      await render({ active: true, mirrored: true })
      expect(wrapper.className).toBe('video-container active mirrored')
    })
  })

  describe('componentDidUpdate', () => {
    describe('src', () => {
      beforeEach(async () => {
        await render()
        delete video.videoRef.current!.srcObject
      })
      it('updates src only when changed', () => {
        mediaStream = new MediaStream()
        component.setState({
          stream: { url: 'test', stream: mediaStream, userId: '' },
        })
        expect(video.videoRef.current!.src).toBe('http://localhost/test')
        component.setState({
          stream: { url: 'test', stream: mediaStream, userId: '' },
        })
      })
      it('updates srcObject only when changed', () => {
        video.videoRef.current!.srcObject = null
        mediaStream = new MediaStream()
        component.setState({
          stream: { url: 'test', stream: mediaStream, userId: '' },
        })
        expect(video.videoRef.current!.srcObject).toBe(mediaStream)
        component.setState({
          stream: { url: 'test', stream: mediaStream, userId: '' },
        })
      })
    })
  })

})
