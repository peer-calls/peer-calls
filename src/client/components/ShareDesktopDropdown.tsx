import classnames from 'classnames'
import React from 'react'
import { IconType } from 'react-icons'
import { MdRadioButtonChecked, MdRadioButtonUnchecked } from 'react-icons/md'
import { getDesktopStream } from '../actions/MediaActions'
import { removeLocalStream } from '../actions/StreamActions'
import { LocalStream } from '../reducers/streams'
import { Backdrop } from './Backdrop'
import { ToolbarButton } from './ToolbarButton'

export interface ShareDesktopConfig {
  video: true
  audio: boolean
}

const configDesktopOnly: ShareDesktopConfig = {
  audio: false,
  video: true,
}

const configDesktopAudioVideo: ShareDesktopConfig = {
  audio: true,
  video: true,
}

export interface ShareDesktopDropdownProps {
  className: string
  icon: IconType
  offIcon: IconType
  title: string

  desktopStream: LocalStream | undefined
  onGetDesktopStream: typeof getDesktopStream
  onRemoveLocalStream: typeof removeLocalStream
}

export interface ShareDesktopDropdownState {
  open: boolean
  shareConfig: ShareDesktopConfig | false
}

export class ShareDesktopDropdown extends
React.PureComponent<ShareDesktopDropdownProps, ShareDesktopDropdownState> {
  state: ShareDesktopDropdownState = {
    open: false,
    shareConfig: false,
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
  handleShareDesktop = (shareConfig: ShareDesktopConfig | false) => {
    this.close()

    const { desktopStream } = this.props

    if (desktopStream) {
      const { stream, type } = desktopStream
      this.props.onRemoveLocalStream(stream, type)
    }


    this.setState({
      shareConfig,
    })

    if (!shareConfig) {
      return
    }

    this.props.onGetDesktopStream(shareConfig).catch(() => {
      this.setState({
        shareConfig: false,
      })
    })
  }
  render() {
    const { shareConfig } = this.state

    const classNames = classnames(
      'stream-desktop-menu dropdown-list dropdown-center',
      {
        'dropdown-list-open': this.state.open,
      },
    )

    return (
      <div className='dropdown'>
        <ToolbarButton
          className={this.props.className}
          icon={this.props.icon}
          offIcon={this.props.offIcon}
          on={shareConfig !== false}
          onClick={this.toggleOpen}
          title={this.props.title}
        />
        <Backdrop visible={this.state.open} onClick={this.close} />
        <ul className={classNames}>
          <DesktopShareOption
            config={false}
            name={'Off'}
            onClick={this.handleShareDesktop}
            selected={shareConfig === false}
          />
          <DesktopShareOption
            config={configDesktopAudioVideo}
            name={'Screen with Audio'}
            onClick={this.handleShareDesktop}
            selected={shareConfig === configDesktopAudioVideo}
          />
          <DesktopShareOption
            config={configDesktopOnly}
            name={'Screen only'}
            onClick={this.handleShareDesktop}
            selected={shareConfig === configDesktopOnly}
          />
        </ul>
      </div>
    )
  }
}

export interface DesktopShareOptionProps {
  selected: boolean
  name: string

  config: ShareDesktopConfig | false
  onClick: (config: ShareDesktopConfig | false) => void
}

class DesktopShareOption extends React.PureComponent<DesktopShareOptionProps> {
  handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    this.props.onClick(this.props.config)
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
