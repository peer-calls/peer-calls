import React from 'react'
import classnames from 'classnames'
import { IconType } from 'react-icons'
import { AudioConstraint, VideoConstraint, enumerateDevices, getMediaStream, MediaDevice, setAudioConstraint, setVideoConstraint, FacingConstraint, DeviceConstraint } from '../actions/MediaActions'
import { ToolbarButton } from './ToolbarButton'
import { State } from '../store'
import { MdVideocam, MdMic, MdVideocamOff, MdMicOff } from 'react-icons/md'
import { connect } from 'react-redux'

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
  getMediaStream: typeof getMediaStream

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

export class DeviceDropdown
extends React.PureComponent<DeviceDropdownProps, DeviceDropdownState> {
  state: DeviceDropdownState = {
    open: false,
  }
  handleOpen = () => {
    this.setState({ open: !this.state.open })
  }
  handleDevice = async (constraint: AudioConstraint | VideoConstraint) => {
    this.setState({ open: false })
    let { audioinput: audio, videoinput: video } = this.props

    if (this.props.kind === 'audioinput') {
      audio = constraint as AudioConstraint
      this.props.setAudioConstraint(audio)
    } else {
      video = constraint as VideoConstraint
      this.props.setVideoConstraint(video)
    }

    await this.props.getMediaStream({ audio, video })
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
          onClick={this.handleOpen}
          title={this.props.title}
        />
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
  handleClick = () => {
    this.props.onClick(this.props.device)
  }
  render() {
    const checked = this.props.selected ? '✓' : ''
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
  getMediaStream,
  setVideoConstraint,
  setAudioConstraint,
}

export const AudioDropdown =
  connect(mapAudioStateToProps, avDispatch)(DeviceDropdown)

export const VideoDropdown =
  connect(mapVideoStateToProps, avDispatch)(DeviceDropdown)
