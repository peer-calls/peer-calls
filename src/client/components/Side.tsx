import React from 'react'
import classnames from 'classnames'

export type Left = { left: true }
export type Right = { right: true }
export type Top = { top: true }
export type Bottom = { bottom: true }

export type SideProps = (Left | Right | Top | Bottom) & {
  className?: string
  zIndex: number
  children: React.ReactNode
  align?: 'baseline' | 'center' | 'flex-end'
}

export const Side = React.memo(
  function Side(props: SideProps) {
    const { className, zIndex, ...otherProps } = props
    return (
      <div
        className={classnames('side', className, { ...otherProps })}
        style={{alignItems: props.align  || 'center', zIndex }}
      >
        {props.children}
      </div>
    )
  },
)
