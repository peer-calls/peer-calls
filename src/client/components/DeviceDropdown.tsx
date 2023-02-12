import classnames from 'classnames'
import _debug from 'debug'
import isEqual from 'lodash/isEqual'
import React, { Context } from 'react'
import { IconType } from 'react-icons'
import { MdArrowDropUp, MdMic, MdMicOff, MdRadioButtonChecked, MdRadioButtonUnchecked, MdVideocam, MdVideocamOff } from 'react-icons/md'
import { connect, ReactReduxContext, ReactReduxContextValue } from 'react-redux'
import { AnyAction } from 'redux'
import { enableMediaTrack, getBlankVideoTrack, getDeviceId, getMediaTrack, getTracksByKind, MediaDevice, MediaKind, setDeviceIdOrDisable, setSizeConstraint, SizeConstraint, MediaKindVideo, MediaKindAudio, GetMediaTrackParams } from '../actions/MediaActions'
import { DEVICE_DEFAULT_ID, DEVICE_DISABLED_ID } from '../constants'
import { MediaConstraint } from '../reducers/media'
import { LocalStream } from '../reducers/streams'
import { State, Store } from '../store'
import { Backdrop } from './Backdrop'
import { ToolbarButton } from './ToolbarButton'

const debug = _debug('peercalls')

export interface DeviceDropdownProps {
  className: string
  icon: IconType
  offIcon: IconType
  devices: MediaDevice[]
  title: string
  dropdownTitle: string
  kind: MediaKind
  cameraStream?: LocalStream

  mediaConstraint: MediaConstraint

  getMediaTrack: typeof getMediaTrack
  getBlankVideoTrack: typeof getBlankVideoTrack
  enableMediaTrack: typeof enableMediaTrack

  setDeviceId: typeof setDeviceIdOrDisable
  setSizeConstraint: typeof setSizeConstraint
}

const labels = {
  audio: 'Audio',
  video: 'Video',
}

const qualityLow: SizeConstraint  = {
  width: 320,
  height: 240,
}

const qualityMd: SizeConstraint  = {
  width: 640,
  height: 480,
}

const qualitySd: SizeConstraint  = {
  width: 1280,
  height: 720,
}

const qualityHd: SizeConstraint  = {
  width: 1920,
  height: 1080,
}

export interface DeviceDropdownState {
  open: boolean
}

export class DeviceDropdown
extends React.PureComponent<DeviceDropdownProps, DeviceDropdownState> {
  // hacky way to access the store.getState()
  static contextType: Context<ReactReduxContextValue<Store, AnyAction>> =
    ReactReduxContext

  state: DeviceDropdownState = {
    open: false,
  }
  toggleOpen = (e: React.SyntheticEvent) => {
    e.stopPropagation()
    this.setOpen(!this.state.open)
  }
  toggleDevice = async (e: React.SyntheticEvent) => {
    e.stopPropagation()

    const { mediaConstraint } = this.props

    const shouldEnable = !mediaConstraint.enabled

    const deviceId = getDeviceId(
      shouldEnable,
      mediaConstraint.constraints,
    )

    await this.handleDevice(deviceId)
  }
  close = () => {
    this.setOpen(false)
  }
  setOpen = (open: boolean) => {
    this.setState({ open })
  }
  handleDevice = async (deviceId: string) => {
    this.close()

    const { kind } = this.props

    this.props.setDeviceId({
      kind,
      deviceId,
    })

    await this.getMediaTrack()
  }
  handleQuality = async (quality: SizeConstraint) => {
    this.close()

    const { kind } = this.props

    if (kind !== 'video') {
      return
    }

    const existing = this.props.mediaConstraint.constraints

    const { width, height } = quality

    if (existing.width === width && existing.height === height) {
      // Nothing to do.
      return
    }

    this.props.setSizeConstraint(quality)

    await this.getMediaTrack()
  }
  async getMediaTrack() {
    const { kind, cameraStream } = this.props

    const oldState = this.props.mediaConstraint

    const store = this.context.store as Store
    const newState = store.getState().media[kind]

    // Check if there is already a track of the same kind in our local camera
    // stream.
    const hasExistingTrack = cameraStream &&
      getTracksByKind(cameraStream.stream, kind).length > 0

    if (
      hasExistingTrack &&
      !oldState.enabled &&
      newState.enabled &&
      isEqual(oldState.constraints, newState.constraints)
    ) {
      // Enable the track that was disabled when No <device> was clicked.
      this.props.enableMediaTrack(kind)
      return
    }

    const params: GetMediaTrackParams = {
      constraint: newState.enabled ? newState.constraints : false,
      kind,
    }

    if (
      hasExistingTrack &&
      newState.enabled &&
      kind === 'video'
    ) {
      debug('calling getBlankVideoTrack')
      // Synchronously create a blank media track, which should replace the
      // existing video track and stop it before we request a new video track.
      // This is a workaround for some phones like Samsung Galaxy A52s which
      // sometimes fail to call getUserMedia multiple times. This fixes an
      // issue with Firefox Fenix 109.1.1 crashing on the same device.
      //
      // Since it relies on calling HTMLCanvasElement.createStream, which is
      // experimental and might be missing on some implementations, we catch
      // the error.
      //
      // A better fix would be to simply remove the existing peer track, and
      // add another one, but that would require renegotiation which is slow.
      try {
        this.props.getBlankVideoTrack(params)
      } catch (err) {
        debug('getBlankVideoTrack failed', err)
      }
    }

    await this.props.getMediaTrack(params)
  }
  render() {
    const { mediaConstraint } = this.props

    const devices = this.props.devices

    const { height } = mediaConstraint.constraints

    const classNames = classnames('dropdown-list dropdown-center', {
      'dropdown-list-open': this.state.open,
    })

    const deviceId = getDeviceId(
      mediaConstraint.enabled,
      mediaConstraint.constraints,
    )

    const buttonsRowClassNames = classnames('buttons-row', this.props.className)

    return (
      <div className='dropdown'>
        <div className={buttonsRowClassNames}>
          <ToolbarButton
            className='device-button-toggle'
            icon={this.props.icon}
            offIcon={this.props.offIcon}
            on={mediaConstraint.enabled}
            onClick={this.toggleDevice}
            title={this.props.title}
          />

          <ToolbarButton
            className='device-button-dropdown'
            icon={MdArrowDropUp}
            on={mediaConstraint.enabled}
            onClick={this.toggleOpen}
            title={this.props.dropdownTitle}
          />
        </div>

        <Backdrop visible={this.state.open} onClick={this.close} />
        <ul className={classNames}>
          {this.props.kind === 'video' && (
            <li>
              <ul className='horizontal'>
                <QualityOption
                  onClick={this.handleQuality}
                  constraint={qualityLow}
                  selected={height === qualityLow.height}
                >
                  Lo
                </QualityOption>
                <QualityOption
                  onClick={this.handleQuality}
                  constraint={qualityMd}
                  selected={height === qualityMd.height}
                >
                  Md
                </QualityOption>
                <QualityOption
                  onClick={this.handleQuality}
                  constraint={qualitySd}
                  selected={height === qualitySd.height}
                >
                  Sd
                </QualityOption>
                <QualityOption
                  onClick={this.handleQuality}
                  constraint={qualityHd}
                  selected={height === qualityHd.height}
                >
                  Hd
                </QualityOption>
              </ul>
            </li>
          )}
          <DeviceOption
            deviceId={DEVICE_DISABLED_ID}
            name={'No ' + labels[this.props.kind]}
            onClick={this.handleDevice}
            selected={deviceId === DEVICE_DISABLED_ID}
          />
          <DeviceOption
            deviceId={DEVICE_DEFAULT_ID}
            name={'Default ' + labels[this.props.kind]}
            onClick={this.handleDevice}
            selected={deviceId === DEVICE_DEFAULT_ID}
          />
          {devices.map(device => (
            <DeviceOption
              deviceId={device.id}
              key={device.id}
              name={device.name}
              onClick={this.handleDevice}
              selected={device.id === deviceId}
            />
          ))}
        </ul>
      </div>
    )
  }
}

export interface DeviceOptionProps {
  selected: boolean
  name: string
  deviceId: string
  onClick: (deviceId: string) => void
}

export class DeviceOption extends React.PureComponent<DeviceOptionProps> {
  handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    this.props.onClick(this.props.deviceId)
  }
  render() {
    const checked = this.props.selected
      ? <MdRadioButtonChecked />
      : <MdRadioButtonUnchecked />
    return (
      <li className='device' onClick={this.handleClick}>
        {checked} {this.props.name}
      </li>
    )
  }
}

export interface QualityOptionProps {
  constraint: SizeConstraint
  selected: boolean
  onClick: (constraint: SizeConstraint) => void
}

export class QualityOption extends React.PureComponent<QualityOptionProps> {
  handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    this.props.onClick(this.props.constraint)
  }
  render() {
    const { selected } = this.props

    const className = classnames('quality', { selected })

    return (
      <li className={className} onClick={this.handleClick}>
        {this.props.children}
      </li>
    )
  }
}

function mapVideoStateToProps(state: State) {
  const cameraStream = state.streams.localStreams.camera

  return {
    className: 'video',
    icon: MdVideocam,
    offIcon: MdVideocamOff,
    title: 'Toggle video',
    dropdownTitle: 'Camera',
    kind: MediaKindVideo,
    devices: state.media.devices.video,
    mediaConstraint: state.media.video,
    cameraStream,
  }
}

function mapAudioStateToProps(state: State) {
  const cameraStream = state.streams.localStreams.camera

  return {
    className: 'audio',
    icon: MdMic,
    offIcon: MdMicOff,
    title: 'Microphone',
    dropdownTitle: 'Toggle mic',
    kind: MediaKindAudio,
    devices: state.media.devices.audio,
    mediaConstraint: state.media.audio,
    cameraStream,
  }
}

const avDispatch = {
  getMediaTrack,
  getBlankVideoTrack,
  enableMediaTrack,
  setDeviceId: setDeviceIdOrDisable,
  setSizeConstraint,
}

export const AudioDropdown =
  connect(mapAudioStateToProps, avDispatch)(DeviceDropdown)

export const VideoDropdown =
  connect(mapVideoStateToProps, avDispatch)(DeviceDropdown)
