import React from 'react'

export interface MessageProps {
  className: string
}

export class Message extends React.PureComponent<MessageProps> {
  render() {
    return (
      <div className={this.props.className}>
        {this.props.children}
      </div>
    )
  }
}

