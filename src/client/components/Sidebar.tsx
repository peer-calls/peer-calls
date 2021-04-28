import classnames from 'classnames'
import React from 'react'
import { IconType } from 'react-icons'
import { MdChat, MdClose, MdPeople } from 'react-icons/md'
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
    const { panel } = this.state
    return (
      <div className={classnames('sidebar-container', {
        show: this.props.visible,
      })}>
        <div className='sidebar-header'>
          <div className='sidebar-close' onClick={this.props.onClose}>
            <MdClose />
          </div>
          <div className='sidebar-title'>
            <SidebarButton
              activePanel={panel}
              icon={MdChat}
              label='Chat'
              onClick={this.focusPanel}
              panel={panelChat}
            />
            <SidebarButton
              activePanel={panel}
              icon={MdPeople}
              label='Users'
              onClick={this.focusPanel}
              panel={panelUsers}
            />
          </div>
        </div>
          {panel === panelChat && (
            <Chat
              nicknames={nicknames}
              messages={messages}
              sendFile={sendFile}
              sendText={sendText}
            />
          )}
          {panel === panelUsers && (
            <Users nicknames={nicknames} />
          )}
      </div>
    )
  }
}

interface SidebarButtonProps {
  activePanel: Panel
  icon: IconType
  label: string
  panel: Panel
  onClick: (panel: Panel) => void
}

class SidebarButton extends React.PureComponent<SidebarButtonProps> {
  handleClick = () => {
    this.props.onClick(this.props.panel)
  }
  render() {
    const { activePanel, label, icon, panel } = this.props

    const className = classnames('sidebar-button', {
      active: activePanel === panel,
    })

    return (
      <div
        aria-label={label}
        className={className}
        onClick={this.handleClick}
        role='button'
      >
        {icon} {label}
      </div>
    )
  }
}
