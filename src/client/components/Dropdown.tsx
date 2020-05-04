import React from 'react'
import classnames from 'classnames'
import { Backdrop } from './Backdrop'

export interface DropdownProps {
  label: string | React.ReactElement
  children: React.ReactElement<{onClick: () => void}>[]
}

export interface DropdownState {
  open: boolean
}

export class Dropdown
extends React.PureComponent<DropdownProps, DropdownState> {
  state = { open: false }

  handleClick = () => {
    this.setState({ open: !this.state.open })
  }
  close = () => {
    this.setState({ open: false })
  }
  render() {
    const { handleClick } = this
    const classNames = classnames('dropdown-list', {
      'dropdown-list-open': this.state.open,
    })

    const menu = React.Children.map(
      this.props.children,
      child => {
        const onClick = child.props.onClick
        return React.cloneElement(child, {
          ...child.props,
          onClick: () => {
            handleClick()
            onClick()
          },
        })
      },
    )

    return (
      <div className='dropdown'>
        <button onClick={handleClick} >{this.props.label}</button>
        <Backdrop onClick={this.close} visible={this.state.open} />
        <ul className={classNames}>
          {menu}
        </ul>
      </div>
    )
  }
}
