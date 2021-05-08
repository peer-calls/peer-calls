import classnames from 'classnames'
import React from 'react'
import { MdClose } from 'react-icons/md'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { Message } from '../reducers/messages'
import { Nicknames } from '../reducers/nicknames'
import Chat from './Chat'
import Users from './Users'

export interface SidebarProps {
  visible: boolean
  onClose: () => void

  // Chat
  messages: Message[]
  nicknames: Nicknames
  sendFile: (file: File) => void
  sendText: (message: string) => void

  // Users
  play: () => void
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
}

export interface SidebarState {
  panel: Panel
}

type Panel = 'chat' | 'users'

const panelChat: Panel = 'chat'
const panelUsers: Panel = 'users'

export default class Sidebar
extends React.PureComponent<SidebarProps, SidebarState> {
  state: SidebarState = {
    panel: 'chat',
  }
  focusPanel = (panel: Panel) => {
    this.setState({
      panel,
    })
  }
  render () {
    const { messages, nicknames, sendFile, sendText } = this.props
    const { onMinimizeToggle } = this.props
    const { panel } = this.state
    return (
      <div className={classnames('sidebar', {
        show: this.props.visible,
      })}>
        <div className='sidebar-header'>
          <div className='sidebar-close' onClick={this.props.onClose}>
            <MdClose />
          </div>
          <ul className='sidebar-menu'>
            <SidebarButton
              activePanel={panel}
              className='sidebar-menu-chat'
              label='Chat'
              onClick={this.focusPanel}
              panel={panelChat}
            />
            <SidebarButton
              activePanel={panel}
              className='sidebar-menu-users'
              label='Users'
              onClick={this.focusPanel}
              panel={panelUsers}
            />
          </ul>
        </div>
        <div className='sidebar-content'>
          {panel === panelChat && (
            <Chat
              nicknames={nicknames}
              messages={messages}
              sendFile={sendFile}
              sendText={sendText}
              visible={true}
            />
          )}
          {panel === panelUsers && (
            <Users
              onMinimizeToggle={onMinimizeToggle}
              play={this.props.play}
            />
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