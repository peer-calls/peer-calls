jest.mock('../window')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { StreamWithURL } from '../reducers/streams'
import { WindowState } from '../reducers/windowStates'
import { MediaStream } from '../window'
import Video, { VideoProps } from './Video'

describe('components/Video', () => {

  interface VideoState {
    stream: null | StreamWithURL
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
        stream={this.state.stream || this.props.stream}
        onMinimizeToggle={this.props.onMinimizeToggle}
        play={this.props.play}
        userId="test"
        muted={this.props.muted}
        mirrored={this.props.mirrored}
        nickname={this.props.nickname}
        windowState={this.props.windowState}
      />
    }
  }

  let component: VideoWrapper
  let video: Video
  let onMinimizeToggle:
    jest.MockedFunction<(payload: MinimizeTogglePayload) => void>
  let mediaStream: MediaStream
  let url: string
  let wrapper: Element
  let nickname: string

  interface Flags {
    muted: boolean
    mirrored: boolean
    windowState: WindowState
  }
  const defaultFlags: Flags = {
    muted: false,
    mirrored: false,
    windowState: undefined,
  }
  async function render (args?: Partial<Flags>) {
    nickname = 'john'
    const flags: Flags = Object.assign({}, defaultFlags, args)
    onMinimizeToggle = jest.fn()
    mediaStream = new MediaStream()
    const div = document.createElement('div')
    component = await new Promise<VideoWrapper>(resolve => {
      const stream: StreamWithURL = {
        stream: mediaStream,
        streamId: mediaStream.id,
        url,
      }
      ReactDOM.render(
        <VideoWrapper
          ref={instance => resolve(instance!)}
          stream={stream}
          play={play}
          userId="test"
          muted={flags.muted}
          mirrored={flags.mirrored}
          onMinimizeToggle={onMinimizeToggle}
          nickname={nickname}
          windowState={flags.windowState}
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

    it('Mirrored and widnowState propogate to rendered classes', async () => {
      await render({ mirrored: true })
      expect(wrapper.className).toBe('video-container mirrored')
    })

    it('Mirrored and windowState propogate to rendered classes', async () => {
      await render({ mirrored: true, windowState: 'minimized' })
      expect(wrapper.className).toBe('video-container minimized mirrored')
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
          stream: {
            url: 'test',
            stream: mediaStream,
            streamId: mediaStream.id,
          },
        })
        expect(video.videoRef.current!.src).toBe('http://localhost/test')
        component.setState({
          stream: {
            url: 'test',
            stream: mediaStream,
            streamId: mediaStream.id,
          },
        })
      })
      it('updates srcObject only when changed', () => {
        video.videoRef.current!.srcObject = null
        mediaStream = new MediaStream()
        component.setState({
          stream: {
            url: 'test',
            stream: mediaStream,
            streamId: mediaStream.id,
          },
        })
        expect(video.videoRef.current!.srcObject).toBe(mediaStream)
        component.setState({
          stream: {
            url: 'test',
            stream: mediaStream,
            streamId: mediaStream.id,
          },
        })
      })
    })
  })

})
