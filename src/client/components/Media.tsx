import React from 'react'
import { connect } from 'react-redux'
import { AudioConstraint, MediaDevice, setAudioConstraint, setVideoConstraint, VideoConstraint } from '../actions/MediaActions'
import { MediaState } from '../reducers/media'
import { State } from '../store'

export type MediaProps = MediaState & {
  onSetVideoConstraint: typeof setVideoConstraint
  onSetAudioConstraint: typeof setAudioConstraint
  onSave: () => void
}

function getId(constraint: VideoConstraint | AudioConstraint) {
  return typeof constraint === 'object' && 'deviceId' in constraint
    ? constraint.deviceId
    : ''
}

function mapStateToProps(state: State) {
  return {
    ...state.media,
  }
}

const mapDispatchToProps = {
  onSetVideoConstraint: setVideoConstraint,
  onSetAudioConstraint: setAudioConstraint,
}

const c = connect(mapStateToProps, mapDispatchToProps)

export const Media = c(React.memo(function Media(props: MediaProps) {

  function onSave(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    props.onSave()
  }

  function onVideoChange(event: React.ChangeEvent<HTMLSelectElement>) {
    const constraint: VideoConstraint = JSON.parse(event.target.value)
    props.onSetVideoConstraint(constraint)
  }

  function onAudioChange(event: React.ChangeEvent<HTMLSelectElement>) {
    const constraint: AudioConstraint = JSON.parse(event.target.value)
    props.onSetAudioConstraint(constraint)
  }

  const videoId = getId(props.video)
  const audioId = getId(props.audio)

  return (
    <form className='media' onSubmit={onSave}>
      <select className='media-video' onChange={onVideoChange} value={videoId}>
        <Options
          devices={props.devices}
          default='{"facingMode":"user"}'
          type='videoinput'
        />
      </select>

      <select className='media-audio' onChange={onAudioChange} value={audioId}>
        <Options
          devices={props.devices}
          default='true'
          type='audioinput'
        />
      </select>

      <button type='submit'>
        Save
      </button>
    </form>
  )
}))

interface OptionsProps {
  devices: MediaDevice[]
  type: 'audioinput' | 'videoinput'
  default: string
}

function Options(props: OptionsProps) {
  return (
    <React.Fragment>
      <option value='false'>Disabled</option>
      <option value={props.default}>Default</option>
      {
        props.devices
        .filter(device => device.type === props.type)
        .map(device =>
          <option
            key={device.id}
            value={JSON.stringify({deviceId: device.id})}>{device.name}
          </option>,
        )
      }
    </React.Fragment>
  )
}
