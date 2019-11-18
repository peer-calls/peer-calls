import Notifications from './Notifications'
import React from 'react'
import ReactDOM from 'react-dom'
import { Notification, NotificationDismissAction } from '../actions/NotifyActions'

describe('Notifications', () => {

  let notifications: Record<string, Notification>
  let dismiss: jest.Mock<NotificationDismissAction, [string]>
  beforeEach(() => {
    notifications = {
      one: {
        id: 'one',
        message: 'test',
        type: 'error',
      },
    }
    dismiss = jest.fn()
  })

  let div: HTMLDivElement
  async function render() {
    div = document.createElement('div')
    return new Promise<Notifications>(resolve => {
      ReactDOM.render(
        <Notifications
          ref={n => resolve(n!)}
          notifications={notifications}
          dismiss={dismiss}
        />,
        div,
      )
    })
  }

  describe('render', () => {
    it('renders', async () => {
      await render()
      ReactDOM.unmountComponentAtNode(div)
    })
  })

})
