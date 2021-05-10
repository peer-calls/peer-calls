import classnames from 'classnames'
import React, {ReactFragment} from 'react'
import { IconType } from 'react-icons'
import { MdRadioButtonChecked, MdRadioButtonUnchecked } from 'react-icons/md'
import { getDesktopStream } from '../actions/MediaActions'
import { removeLocalStream } from '../actions/StreamActions'
import { LocalStream } from '../reducers/streams'
import { Backdrop } from './Backdrop'
import { ToolbarButton } from './ToolbarButton'
import { config } from '../window'
import { RES_IMG_FIREFOX_SHARE } from '../constants'

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
  rejectedShare: boolean
  popupContent: ReactFragment | null
  isDisabled: boolean
}

export class ShareDesktopDropdown extends
React.PureComponent<ShareDesktopDropdownProps, ShareDesktopDropdownState> {

  constructor(props: ShareDesktopDropdownProps) {
    super(props)
    // mobile devices don't support screen sharing
    const isMobile = /Android|iPhone|iPad|iPod/i.test(
      window.navigator.userAgent,
    )
    this.state = {
      open: false,
      shareConfig: false,
      rejectedShare: false,
      popupContent: null,
      isDisabled: isMobile,
    }
  }

  toggleOpen = (e: React.SyntheticEvent) => {
    e.stopPropagation()
    if (!this.state.isDisabled) {
      this.setOpen(!this.state.open)
    }
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

      // Remove onended handler from all tracks. See below for more info.
      stream.getTracks().forEach(t => {
        t.onended = null
      })

      this.props.onRemoveLocalStream(stream, type)
    }

    this.setState({
      shareConfig,
    })

    if (!shareConfig) {
      return
    }

    this.props.onGetDesktopStream(shareConfig)
    .then(payload => {
      const tracks = payload.stream.getTracks()
      let activeTracks = tracks.length

      // Remove the stream after all tracks end. This ensures the "Stop
      // sharing" desktop click in Chrome is handled correctly.
      payload.stream.getTracks().forEach(t => {
        t.onended = () => {
          activeTracks--
          if (activeTracks === 0) {
            this.setState({
              shareConfig: false,
            })

            this.props.onRemoveLocalStream(payload.stream, payload.type)
          }
        }
      })
    })
    .catch(() => {
      const browser = window.navigator.userAgent.toLowerCase()
      if (browser.indexOf('firefox') > -1) {
        this.handleFirefoxRejection()
      }

      this.setState({
        shareConfig: false,
      })
    })
  }

  handleFirefoxRejection() {
    if (!this.state.rejectedShare) {
      return this.setState({rejectedShare: true})
    }

    this.setState({
      popupContent: <>
        <div style={{paddingBottom: '2em'}}>
          If you dismissed a sharing dialog previously,
          you have to remove the resource restriction.<br/>
          Click on site permissions in the address bar
          and remove the blocked resource you want to use.
        </div>
        <img src={config.baseUrl + RES_IMG_FIREFOX_SHARE} />
      </>,
    })
  }

  closePopup = () => {
    this.setState({popupContent: null})
  }

  render() {
    const { shareConfig, popupContent, isDisabled } = this.state

    const classNames = classnames(
      'stream-desktop-menu dropdown-list dropdown-center',
      {
        'dropdown-list-open': this.state.open,
      },
    )

    return (
      <>
        {popupContent && (<div className='popup-overlay'>
          <div className='popup-window'>
            <div onClick={this.closePopup} className='popup-close'>&times;</div>
            <div className='popup-content'>
              {popupContent}
            </div>
          </div>
        </div>)}

        <div className='dropdown'>
          <ToolbarButton
            className={this.props.className + (isDisabled ? ' disabled' : '')}
            icon={this.props.icon}
            offIcon={this.props.offIcon}
            on={shareConfig !== false}
            onClick={this.toggleOpen}
            title={this.props.title}
          />
          <Backdrop visible={this.state.open} onClick={this.close}/>
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
      </>
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
