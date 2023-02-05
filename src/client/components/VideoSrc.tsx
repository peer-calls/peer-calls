import React, { ReactEventHandler } from 'react'
import classnames from 'classnames'

interface VideoSrcProps {
  id?: string
  autoPlay: boolean
  onClick?: ReactEventHandler<HTMLVideoElement>
  onLoadedMetadata?: ReactEventHandler<HTMLVideoElement>
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
