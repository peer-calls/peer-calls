jest.mock('../audio')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { audioProcessor } from '../audio'

import VUMeter from './VUMeter'

describe('components/VUMeter', () => {

  async function setup<P>(Component: React.ComponentClass<P>) {
    class Wrapper extends React.PureComponent<P, P> {
      constructor(props: P) {
        super(props)
        this.state = {...props}
      }

      render() {
        return <Component {...this.state} />
      }
    }

    const div = document.createElement('div')

    const render = async (props: P) => {
      const wrapper = await new Promise<Wrapper>(resolve => {
        ReactDOM.render(
          <Wrapper
            ref={instance => resolve(instance!)}
            {...props}
          />,
          div,
        )
      })

      const component = TestUtils
      .findRenderedComponentWithType(wrapper, Component)

      const node = div.children[0] as HTMLElement

      const setProps = (props: P) => {
        wrapper.setState(props)
      }

      const unmount = () => ReactDOM.unmountComponentAtNode(div)

      return { node, component, setProps, unmount }
    }

    return render
  }


  describe('render', () => {
    it('renders', async () => {
      const render  = await setup(VUMeter)
      const { node, setProps } = await render({ streamId: undefined })
      expect(node).toBeTruthy()
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-0'])

      setProps({ streamId: 'alpha' })

      const subMock = audioProcessor
      .subscribe as jest.MockedFunction<typeof audioProcessor.subscribe>

      expect(subMock.mock.calls).toEqual([[ 'alpha', jasmine.any(Function) ]])

      const callback = subMock.mock.calls[0][1]

      callback({ type: 'volume', volume: 1 })
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-5'])

      callback({ type: 'volume', volume: 0.4 })
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-4'])

      callback({ type: 'volume', volume: 0.3 })
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-3'])

      callback({ type: 'volume', volume: 0.2 })
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-2'])

      callback({ type: 'volume', volume: 0.1 })
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-1'])

      callback({ type: 'volume', volume: 0 })
      expect(Array.from(node.classList))
      .toEqual(['vu-meter', 'vu-meter-level-0'])
    })
  })

})
