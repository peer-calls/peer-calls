import React from 'react'
import classnames from 'classnames'

export type Left = { left: true }
export type Right = { right: true }
export type Top = { top: true }
export type Bottom = { bottom: true }

export type SideProps = (Left | Right | Top | Bottom) & {
  zIndex: number
  children: React.ReactNode
  align?: 'baseline' | 'center' | 'end'
}

export const Side = React.memo(
  function Side(props: SideProps) {
    const className = classnames('side', { ...props })
    return (
      <div
        className={className}
        style={{alignItems: props.align  || 'center', zIndex: props.zIndex}}
      >
        {props.children}
      </div>
    )
  },
)
