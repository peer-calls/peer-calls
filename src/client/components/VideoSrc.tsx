import classnames from 'classnames'
import React, { ReactEventHandler } from 'react'
import { Dim } from '../frame'

export interface VideoSrcProps {
  id?: string
  autoPlay: boolean
  onClick?: ReactEventHandler<HTMLVideoElement>
  onLoadedMetadata?: ReactEventHandler<HTMLVideoElement>
  onResize?: (dimensions: Dim) => void
  src?: string
  srcObject: MediaStream | null
  muted?: boolean
  objectFit?: string
  mirrored?: boolean
}

const videoStyle = {
  width: '100%',
}

export default class VideoSrc extends React.PureComponent<VideoSrcProps> {
  videoRef = React.createRef<HTMLVideoElement>()

  componentDidMount () {
    this.componentDidUpdate()

    this.videoRef.current!.onresize = e => {
      const el = e.target as HTMLVideoElement
      this.maybeTriggerResize(el)
    }
  }
  componentDidUpdate() {
    const { srcObject, src } = this.props
    const muted = !!this.props.muted

    const video = this.videoRef.current

    if (video) {
      if ('srcObject' in video as unknown) {
        if (video.srcObject !== srcObject) {
          video.srcObject = srcObject
        }
      } else if (video.src !== src) {
        video.src = src || ''
      }

      // Rather than setting muted property in <video> directly, we set it here
      // to fix some issues in tests. For more details see commit 4b3cf45bf.
      video.muted = muted

      video.style.objectFit = this.props.objectFit || ''
    }
  }
  handleLoadedMetadata = (e: React.SyntheticEvent<HTMLVideoElement>) => {
    const el = e.target as HTMLVideoElement
    this.maybeTriggerResize(el)

    if (this.props.onLoadedMetadata) {
      this.props.onLoadedMetadata(e)
    }
  }
  maybeTriggerResize = (el: HTMLVideoElement) => {
    const { onResize } = this.props

    if (onResize && el.videoWidth && el.videoHeight) {
      onResize({
        x: el.videoWidth,
        y: el.videoHeight,
      })
    }
  }
  render() {
    const { mirrored } = this.props

    const className = classnames({
      mirrored,
    })

    return (
      <video
        id={this.props.id}
        className={className}
        autoPlay={this.props.autoPlay}
        onClick={this.props.onClick}
        onLoadedMetadata={this.props.onLoadedMetadata}
        playsInline={true}
        ref={this.videoRef}
        style={videoStyle}
      />
    )
  }
}
