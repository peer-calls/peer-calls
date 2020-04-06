import React from 'react'
import { connect } from 'react-redux'
import { AudioConstraint, MediaDevice, setAudioConstraint, setVideoConstraint, VideoConstraint, getMediaStream, enumerateDevices, play } from '../actions/MediaActions'
import { MediaState } from '../reducers/media'
import { State } from '../store'
import { Alerts, Alert } from './Alerts'
import { info, warning, error } from '../actions/NotifyActions'
import { ME, STREAM_TYPE_CAMERA } from '../constants'
import { dial } from '../actions/CallActions'
import { setNickname } from '../actions/NicknameActions'

export type MediaProps = MediaState & {
  onDial: () => void
  visible: boolean
  enumerateDevices: typeof enumerateDevices
  onSetVideoConstraint: typeof setVideoConstraint
  onSetAudioConstraint: typeof setAudioConstraint
  getMediaStream: typeof getMediaStream
  play: typeof play
  logInfo: typeof info
  logWarning: typeof warning
  logError: typeof error
  onSetNickname: typeof setNickname
  nickname?: string
}

function mapStateToProps(state: State) {
  const localStream = state.streams[ME]
  const hidden = !!localStream &&
    localStream.streams.filter(s => s.type === STREAM_TYPE_CAMERA).length > 0
  const visible = !hidden
  return {
    ...state.media,
    nickname: state.nicknames[ME],
    visible,
  }
}

const mapDispatchToProps = {
  enumerateDevices,
  onDial: dial,
  onSetVideoConstraint: setVideoConstraint,
  onSetAudioConstraint: setAudioConstraint,
  onSetNickname: setNickname,
  getMediaStream,
  play,
  logInfo: info,
  logWarning: warning,
  logError: error,
}

const c = connect(mapStateToProps, mapDispatchToProps)

export const MediaForm = React.memo(function MediaForm(props: MediaProps) {
  if (!props.visible) {
    return null
  }

  React.useMemo(async () => await props.enumerateDevices(), [])

  async function onSave(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const { audio, video } = props
    try {
      await props.getMediaStream({ audio, video })
    } catch (err) {
      props.logError('Error: {0}', err)
    } finally {
      props.logInfo('Dialling...')
      props.onDial()
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
      <input
        type='text'
        onChange={e => props.onSetNickname({
          nickname: e.target.value,
          userId: ME,
        })}
        value={props.nickname}
      />

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
})

export interface AutoplayProps {
  play: () => void
}

export const AutoplayMessage = React.memo(
  function Autoplay(props: AutoplayProps) {
    return (
      <React.Fragment>
        Your browser has blocked video autoplay on this page.
        To continue with your call, please press the play button:
        &nbsp;
        <button className='button' onClick={props.play}>
          Play
        </button>
      </React.Fragment>
    )
  },
)

export const Media = c(React.memo(function Media(props: MediaProps) {
  return (
    <div className='media-container'>
      <Alerts>
        {props.autoplayError && (
          <Alert>
            <AutoplayMessage play={props.play} />
          </Alert>
        )}
      </Alerts>

      <MediaForm {...props} />
    </div>
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
      <option value='false'>Disable {label}</option>
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
