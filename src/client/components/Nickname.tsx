import React from 'react'
import { NicknameMessage } from '../actions/PeerActions'
import { nickname } from '../window'

export interface NicknameProps {
  value: string
  onChange: (message: NicknameMessage) => void
  localUser?: boolean
}

export class Nickname extends React.PureComponent<NicknameProps> {
  render() {
    if (this.props.localUser) {
      return (
        <MemoEditableNickname
          value={this.props.value}
          onChange={this.props.onChange}
        />
      )
    }
    return <ReadOnlyNickname value={this.props.value} />
  }
}

interface EditableNicknameProps {
  value: string
  onChange: (message: NicknameMessage) => void
}

const MemoEditableNickname = React.memo(EditableNickname)

function EditableNickname(props: EditableNicknameProps) {

  const [value, setValue] = React.useState(props.value)
  const handleChange =
    (e: React.ChangeEvent<HTMLInputElement>) => setValue(e.target.value)

  function handleKeyPress(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      e.currentTarget.blur()
    }
  }

  function update() {
    let nick = value
    if (!value) {
      setValue(nickname)
      nick = nickname
    }
    props.onChange({
      type: 'nickname',
      payload: { nickname: nick },
    })
  }

  return (
    <input
      className="nickname"
      type="text"
      onChange={handleChange}
      onKeyPress={handleKeyPress}
      onBlur={update}
      value={value}
    />
  )
}

interface ReadOnlyNicknameProps {
  value: string
}

function ReadOnlyNickname(props: ReadOnlyNicknameProps) {
  return <span className="nickname">{props.value}</span>
}
