import React from 'react'
import { connect } from 'react-redux'
import { AudioConstraint, MediaDevice, setAudioConstraint, setVideoConstraint, VideoConstraint, getMediaStream, enumerateDevices } from '../actions/MediaActions'
import { MediaState } from '../reducers/media'
import { State } from '../store'

export type MediaProps = MediaState & {
  enumerateDevices: typeof enumerateDevices
  onSetVideoConstraint: typeof setVideoConstraint
  onSetAudioConstraint: typeof setAudioConstraint
  getMediaStream: typeof getMediaStream
}

function mapStateToProps(state: State) {
  return {
    ...state.media,
  }
}

const mapDispatchToProps = {
  enumerateDevices,
  onSetVideoConstraint: setVideoConstraint,
  onSetAudioConstraint: setAudioConstraint,
  getMediaStream,
}

const c = connect(mapStateToProps, mapDispatchToProps)

export const Media = c(React.memo(function Media(props: MediaProps) {

  React.useMemo(async () => await props.enumerateDevices(), [])

  async function onSave(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const { audio, video } = props
    try {
      await props.getMediaStream({ audio, video })
    } catch (err) {
      console.error(err.stack)
      // TODO display a message
    }
  }

  function onVideoChange(event: React.ChangeEvent<HTMLSelectElement>) {
    const constraint: VideoConstraint = JSON.parse(event.target.value)
    props.onSetVideoConstraint(constraint)
  }

  function onAudioChange(event: React.ChangeEvent<HTMLSelectElement>) {
    const constraint: AudioConstraint = JSON.parse(event.target.value)
    props.onSetAudioConstraint(constraint)
  }

  const videoId = JSON.stringify(props.video)
  const audioId = JSON.stringify(props.audio)

  return (
    <form className='media' onSubmit={onSave}>
      <select
        name='video-input'
        onChange={onVideoChange}
        value={videoId}
      >
        <Options
          devices={props.devices}
          default='{"facingMode":"user"}'
          type='videoinput'
        />
      </select>

      <select
        name='audio-input'
        onChange={onAudioChange}
        value={audioId}
      >
        <Options
          devices={props.devices}
          default='true'
          type='audioinput'
        />
      </select>

      <button type='submit'>
        Join Call
      </button>
    </form>
  )
}))

interface OptionsProps {
  devices: MediaDevice[]
  type: 'audioinput' | 'videoinput'
  default: string
}

const labels = {
  audioinput: 'Audio',
  videoinput: 'Video',
}

function Options(props: OptionsProps) {
  const label = labels[props.type]
  return (
    <React.Fragment>
      <option value='false'>{label} disabled</option>
      <option value={props.default}>Default {label}</option>
      {
        props.devices
        .filter(device => device.type === props.type)
        .map(device =>
          <option
            key={device.id}
            value={JSON.stringify({deviceId: device.id})}
          >
            {device.name || device.type}
          </option>,
        )
      }
    </React.Fragment>
  )
}
