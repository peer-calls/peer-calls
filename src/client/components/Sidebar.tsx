import classnames from 'classnames'
import React from 'react'
import { MdClose } from 'react-icons/md'
import { Panel, sidebarPanelChat, sidebarPanelSettings, sidebarPanelUsers } from '../actions/SidebarActions'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { Message } from '../reducers/messages'
import { Nicknames } from '../reducers/nicknames'
import Chat from './Chat'
import Settings from './Settings'
import Users from './Users'

export interface SidebarProps {
  // Panel state
  onHide: () => void
  onShow: (panel: Panel) => void
  visible: boolean
  panel: Panel

  // Chat
  messages: Message[]
  nicknames: Nicknames
  sendFile: (file: File) => void
  sendText: (message: string) => void

  // Users
  play: () => void
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
}

export default class Sidebar extends React.PureComponent<SidebarProps> {
  render () {
    const { messages, nicknames, sendFile, sendText, panel } = this.props
    const { onMinimizeToggle } = this.props
    return (
      <div className={classnames('sidebar', {
        show: this.props.visible,
      })}>
        <div className='sidebar-header'>
          <div className='sidebar-close' onClick={this.props.onHide}>
            <MdClose />
          </div>
          <ul className='sidebar-menu'>
            <SidebarButton
              activePanel={panel}
              className='sidebar-menu-chat'
              label='Chat'
              onClick={this.props.onShow}
              panel={sidebarPanelChat}
            />
            <SidebarButton
              activePanel={panel}
              className='sidebar-menu-users'
              label='Users'
              onClick={this.props.onShow}
              panel={sidebarPanelUsers}
            />
            <SidebarButton
              activePanel={panel}
              className='sidebar-menu-settings'
              label='Settings'
              onClick={this.props.onShow}
              panel={sidebarPanelSettings}
            />
          </ul>
        </div>
        <div className='sidebar-content'>
          {panel === sidebarPanelChat && (
            <Chat
              nicknames={nicknames}
              messages={messages}
              sendFile={sendFile}
              sendText={sendText}
              visible={this.props.visible}
            />
          )}
          {panel === sidebarPanelUsers && (
            <Users
              onMinimizeToggle={onMinimizeToggle}
              play={this.props.play}
            />
          )}
          {panel === sidebarPanelSettings && (
            <Settings />
          )}
        </div>
      </div>
    )
  }
}

interface SidebarButtonProps {
  activePanel: Panel
  className?: string
  label: string
  panel: Panel
  onClick: (panel: Panel) => void
}

class SidebarButton extends React.PureComponent<SidebarButtonProps> {
  handleClick = () => {
    this.props.onClick(this.props.panel)
  }
  render() {
    const { activePanel, label, panel } = this.props

    const className = classnames(this.props.className, 'sidebar-button', {
      active: activePanel === panel,
    })

    return (
      <li
        aria-label={label}
        className={className}
        onClick={this.handleClick}
        role='button'
      >
        {label}
      </li>
    )
  }
}
