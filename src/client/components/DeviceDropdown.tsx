import React from 'react'
import classnames from 'classnames'
import { IconType } from 'react-icons'
import { AudioConstraint, VideoConstraint, enumerateDevices, enableMediaTrack, MediaDevice, setAudioConstraint, setVideoConstraint, FacingConstraint, DeviceConstraint, getMediaTrack, GetMediaTrackParams } from '../actions/MediaActions'
import { ToolbarButton } from './ToolbarButton'
import { State } from '../store'
import { MdVideocam, MdMic, MdVideocamOff, MdMicOff, MdRadioButtonChecked, MdRadioButtonUnchecked } from 'react-icons/md'
import { connect } from 'react-redux'
import { Backdrop } from './Backdrop'

export type Constraint = AudioConstraint | VideoConstraint

export interface DeviceDropdownProps {
  className: string
  icon: IconType
  offIcon: IconType
  devices: MediaDevice[]
  title: string
  kind: 'videoinput' | 'audioinput'

  audioinput: AudioConstraint
  videoinput: VideoConstraint

  enumerateDevices: typeof enumerateDevices
  getMediaTrack: typeof getMediaTrack
  enableMediaTrack: typeof enableMediaTrack

  setAudioConstraint: typeof setAudioConstraint
  setVideoConstraint: typeof setVideoConstraint
}

const labels = {
  audioinput: 'Audio',
  videoinput: 'Video',
}

const defaultDevices = {
  audioinput: true,
  videoinput: {facingMode: 'user'} as FacingConstraint,
}

export interface DeviceDropdownState {
  open: boolean
}

function isDeviceConstraint(d: Constraint): d is DeviceConstraint {
  return typeof d === 'object' && 'deviceId' in d && !!d.deviceId
}

// Remove "name" from constraint so we can compare using stringify
function toPlainConstraint(c: Constraint): Constraint{
  if (isDeviceConstraint(c)) {
    return { deviceId: c.deviceId }
  }
  return c
}

export class DeviceDropdown
extends React.PureComponent<DeviceDropdownProps, DeviceDropdownState> {
  private lastDevice?: AudioConstraint | VideoConstraint

  state: DeviceDropdownState = {
    open: false,
  }
  toggleOpen = (e: React.SyntheticEvent) => {
    e.stopPropagation()
    this.setOpen(!this.state.open)
  }
  close = () => {
    this.setOpen(false)
  }
  setOpen = (open: boolean) => {
    this.setState({ open })
  }
  handleDevice = async (constraint: AudioConstraint | VideoConstraint) => {
    this.close()
    let { audioinput: audio, videoinput: video } = this.props

    if (this.props.kind === 'audioinput') {
      audio = toPlainConstraint(constraint) as AudioConstraint

      if (!audio && this.props.audioinput) {
        this.lastDevice = this.props.audioinput
      }

      this.props.setAudioConstraint(audio)
      await this.getMediaTrack({
        constraint: audio,
        kind: 'audio',
      })
    } else {
      video = toPlainConstraint(constraint) as VideoConstraint

      if (!video && this.props.videoinput) {
        this.lastDevice = this.props.videoinput
      }

      this.props.setVideoConstraint(video)
      await this.getMediaTrack({
        constraint: video,
        kind: 'video',
      })
    }
  }
  async getMediaTrack(
    params: GetMediaTrackParams,
  ) {
    if (
      params.constraint &&
      JSON.stringify(params.constraint) === JSON.stringify(this.lastDevice)
    ) {
      // enable the track that was disabled when No <device> was clicked
      this.props.enableMediaTrack(params.kind)
      return
    }

    await this.props.getMediaTrack(params)
  }
  render() {
    const selectedDevice: Constraint = this.props[this.props.kind]

    const devices = this.props.devices
    .filter(device => device.type === this.props.kind)
    .map(device => ({
      deviceId: device.id,
      name: device.name,
    }))


    const classNames = classnames('dropdown-list dropdown-center', {
      'dropdown-list-open': this.state.open,
    })

    return (
      <div className='dropdown'>
        <ToolbarButton
          className={this.props.className}
          icon={this.props.icon}
          offIcon={this.props.offIcon}
          on={selectedDevice === false}
          onClick={this.toggleOpen}
          title={this.props.title}
        />
        <Backdrop visible={this.state.open} onClick={this.close} />
        <ul className={classNames}>
          <DeviceOption
            device={false}
            name={'No ' + labels[this.props.kind]}
            onClick={this.handleDevice}
            selected={selectedDevice === false}
          />
          <DeviceOption
            device={defaultDevices[this.props.kind]}
            name={'Default ' + labels[this.props.kind]}
            onClick={this.handleDevice}
            selected={
              JSON.stringify(selectedDevice) ===
                JSON.stringify(defaultDevices[this.props.kind])
            }
          />
          {devices.map(device => (
            <DeviceOption
              device={device}
              key={device.deviceId}
              name={device.name}
              onClick={this.handleDevice}
              selected={
                isDeviceConstraint(selectedDevice) &&
                selectedDevice.deviceId  === device.deviceId
              }
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
  device: Constraint
  onClick: (device: Constraint) => void
}

export class DeviceOption extends React.PureComponent<DeviceOptionProps> {
  handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    this.props.onClick(this.props.device)
  }
  render() {
    const checked = this.props.selected
      ? <MdRadioButtonChecked />
      : <MdRadioButtonUnchecked />
    return (
      <li onClick={this.handleClick}>
        {checked} {this.props.name}
      </li>
    )
  }
}

function mapVideoStateToProps(state: State) {
  return {
    className: 'video',
    icon: MdVideocamOff,
    offIcon: MdVideocam,
    title: 'Camera',
    kind: 'videoinput' as 'videoinput',
    devices: state.media.devices,
    audioinput: state.media.audio,
    videoinput: state.media.video,
  }
}

function mapAudioStateToProps(state: State) {
  return {
    className: 'audio',
    icon: MdMicOff,
    offIcon: MdMic,
    title: 'Microphone',
    kind: 'audioinput' as 'audioinput',
    devices: state.media.devices,
    audioinput: state.media.audio,
    videoinput: state.media.video,
  }
}

const avDispatch = {
  enumerateDevices,
  getMediaTrack,
  enableMediaTrack,
  setVideoConstraint,
  setAudioConstraint,
}

export const AudioDropdown =
  connect(mapAudioStateToProps, avDispatch)(DeviceDropdown)

export const VideoDropdown =
  connect(mapVideoStateToProps, avDispatch)(DeviceDropdown)
