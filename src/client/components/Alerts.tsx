import React from 'react'

export interface AlertProps {
  children: React.ReactNode
}

export const Alert = React.memo(
  function Alert(props: AlertProps) {
    return (
      <div className='alert'>
        {props.children}
      </div>
    )
  },
)

export interface AlertsProps {
  children: React.ReactNode
}

export const Alerts = React.memo(
  function Alerts(props: AlertsProps) {
    return (
      <div className="alerts">
        {props.children}
      </div>
    )
  },
)
