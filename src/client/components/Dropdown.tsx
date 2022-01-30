import React, { ReactEventHandler } from 'react'
import classnames from 'classnames'
import { Backdrop } from './Backdrop'

export interface DropdownProps {
  label: string | React.ReactElement
  children: React.ReactElement<{onClick: ReactEventHandler<Element>}>[]
  // fixed will make the dropdown menu use fixed positioning instead of
  // absolute. The position will be manually calculated using
  // getBoundingClientRect relative to the dropdown button.
  fixed?: boolean
}

export interface DropdownState {
  open: boolean
  style: React.CSSProperties
}

export class Dropdown
extends React.PureComponent<DropdownProps, DropdownState> {
  state = {
    open: false,
    style: {},
  }
  buttonRef = React.createRef<HTMLButtonElement>()
  menuRef= React.createRef<HTMLUListElement>()

  handleClick = () => {
    const style: React.CSSProperties = {}

    if (this.props.fixed) {
      const buttonRect = this.buttonRef.current!.getBoundingClientRect()
      const menuRect = this.menuRef.current!.getBoundingClientRect()

      let top = buttonRect.top - menuRect.height
      let left = buttonRect.right - menuRect.width

      // If there's no room at the top, move the menu below.
      if (top < 0) {
        top = buttonRect.bottom
      }

      // If there's no more room to the left, move the menu to the right.
      if (left < 0) {
        left = buttonRect.left
      }

      style.top = top
      style.left = left
    }

    this.setState({
      open: !this.state.open,
      style,
    })
  }
  close = () => {
    this.setState({ open: false })
  }
  render() {
    const { handleClick } = this
    const classNames = classnames('dropdown-list', {
      'dropdown-list-fixed': this.props.fixed,
      'dropdown-list-open': this.state.open,
    })

    const menu = React.Children.map(
      this.props.children,
      child => {
        const onClick = child.props.onClick
        return React.cloneElement(child, {
          ...child.props,
          onClick: (e: React.SyntheticEvent<Element>) => {
            e.preventDefault()

            handleClick()
            onClick(e)
          },
        })
      },
    )

    return (
      <div className='dropdown'>
        <button onClick={handleClick} ref={this.buttonRef}>
          {this.props.label}
        </button>
        <Backdrop onClick={this.close} visible={this.state.open} />
        <ul className={classNames} style={this.state.style} ref={this.menuRef}>
          {menu}
        </ul>
      </div>
    )
  }
}
