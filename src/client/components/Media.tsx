import React from 'react'
import { connect } from 'react-redux'
import { AudioConstraint, MediaDevice, setAudioConstraint, setVideoConstraint, VideoConstraint, getMediaStream, enumerateDevices, play } from '../actions/MediaActions'
import { MediaState } from '../reducers/media'
import { State } from '../store'
import { Alerts, Alert } from './Alerts'
import { info, warning, error } from '../actions/NotifyActions'
import { ME, DialState, DIAL_STATE_HUNG_UP } from '../constants'
import { dial } from '../actions/CallActions'
import { network } from '../window'
import classnames from 'classnames'
import { Unsupported } from './Unsupported'
import { Message } from './Message'
import { MdError } from 'react-icons/md'

export type MediaProps = MediaState & {
  joinEnabled: boolean
  dial: typeof dial
  dialState: DialState
  visible: boolean
  enumerateDevices: typeof enumerateDevices
  onSetVideoConstraint: typeof setVideoConstraint
  onSetAudioConstraint: typeof setAudioConstraint
  getMediaStream: typeof getMediaStream
  play: typeof play
  logInfo: typeof info
  logWarning: typeof warning
  logError: typeof error
  nickname?: string
}

export interface MediaComponentState {
  nickname: string
  error?: boolean
}

function mapStateToProps(state: State) {
  return {
    ...state.media,
    nickname: state.nicknames[ME],
    joinEnabled:
      state.media.dialState === DIAL_STATE_HUNG_UP &&
      state.media.socketConnected &&
      !state.media.loading,
    visible: state.media.dialState === DIAL_STATE_HUNG_UP,
  }
}

const mapDispatchToProps = {
  enumerateDevices,
  dial,
  onSetVideoConstraint: setVideoConstraint,
  onSetAudioConstraint: setAudioConstraint,
  getMediaStream,
  play,
  logInfo: info,
  logWarning: warning,
  logError: error,
}

const c = connect(mapStateToProps, mapDispatchToProps)

export class MediaForm
extends React.PureComponent<MediaProps, MediaComponentState> {
  constructor(props: MediaProps) {
    super(props)
    this.state = {
      nickname: props.nickname || '',
    }
  }

  componentDidMount() {
    this.props.enumerateDevices()
  }
  handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    const { nickname } = this.state
    localStorage && (localStorage.nickname = nickname)
    event.preventDefault()
    const { props } = this
    const { audio, video } = props
    try {
      await props.getMediaStream({ audio, video })
    } catch (err) {
      this.setState({ error: true })
      return
    }
    this.setState({ error: false })

    props.logInfo('Dialling...')
    try {
      await props.dial({
        nickname,
      })
    } catch (err) {
      props.logError('Error dialling: {0}', err)
    }
  }
  handleVideoChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const constraint: VideoConstraint = JSON.parse(event.target.value)
    this.props.onSetVideoConstraint(constraint)
  }

  handleAudioChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    const constraint: AudioConstraint = JSON.parse(event.target.value)
    this.props.onSetAudioConstraint(constraint)
  }
  handleNicknameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ nickname: event.target.value })
  }
  render() {
    const { props } = this
    const { nickname } = this.state
    if (!props.visible) {
      return null
    }

    const videoId = JSON.stringify(props.video)
    const audioId = JSON.stringify(props.audio)

    return (
      <form className='media' onSubmit={this.handleSubmit}>
        <div className='form-item'>
          <label className={classnames({ 'label-error': !nickname })}>
            Enter your name
          </label>
          <input
            required
            className={classnames({error: !nickname})}
            name='nickname'
            type='text'
            placeholder='Name'
            autoFocus
            onChange={this.handleNicknameChange}
            value={nickname}
          />
        </div>

        <div className='form-item'>
          <select
            name='video-input'
            onChange={this.handleVideoChange}
            value={videoId}
            autoComplete='off'
          >
            <Options
              devices={props.devices}
              default='{"facingMode":"user"}'
              type='videoinput'
            />
          </select>
        </div>

        <div className='form-item'>
          <select
            name='audio-input'
            onChange={this.handleAudioChange}
            value={audioId}
            autoComplete='off'
          >
            <Options
              devices={props.devices}
              default='true'
              type='audioinput'
            />
          </select>
        </div>

        <button type='submit' disabled={!props.joinEnabled}>
          Join Call
        </button>

        {this.state.error && (
          <Message className='message-error'>
            <MdError className='icon' />
            <span>
              Could not get access to microphone or camera. Please grant the
              necessary permissions and try again.
            </span>
          </Message>
        )}

        <Unsupported />

        <div className='network-info'>
          <span>Network: {network}</span>
        </div>
      </form>
    )
  }
}

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
      <option value='false'>No {label}</option>
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
