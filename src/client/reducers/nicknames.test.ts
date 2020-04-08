jest.mock('../window')

import { applyMiddleware, createStore, Store } from 'redux'
import { create } from '../middlewares'
import { userId, nickname } from '../window'
import reducers from './index'
import { ME } from '../constants'
import { setNicknames, removeNickname } from '../actions/NicknameActions'
import { getLocalNickname } from '../reducers/nicknames'

describe('reducers/nicknames', () => {

  let store: Store
  beforeEach(() => {
    store = createStore(
      reducers,
      applyMiddleware(...create()),
    )
  })

  describe('defaults', () => {
    it('sets nickname from local store', () => {
      expect(store.getState().nicknames).toEqual({
        [ME]: nickname,
      })
    })
  })

  describe('nicknames set', () => {
    it('sets all nicknames and keeps the local nickname', () => {
      store.dispatch(setNicknames({
        a: 'one',
        b: 'two',
        [userId]: 'three',
      }))
      expect(store.getState().nicknames).toEqual({
        a: 'one',
        b: 'two',
        [ME]: 'three',
      })
    })
  })

  describe('getLocalNickname', () => {

    afterEach(() => {
      localStorage.removeItem('nickname')
    })

    it('reads data from local storage, when available', () => {
      localStorage.setItem('nickname', 'test')
      expect(getLocalNickname()).toBe('test')
    })

    it('reads data from window.nickname as a fallback', () => {
      expect(getLocalNickname()).toBe(nickname)
    })
  })

  describe('removeNickname', () => {

    beforeEach(() => {
      store.dispatch(setNicknames({
        a: 'one',
        b: 'two',
      }))
      expect(store.getState().nicknames).toEqual({
        a: 'one',
        b: 'two',
        [ME]: nickname,
      })
    })

    it('removes a specific nickname', () => {
      store.dispatch(removeNickname({ userId: 'a' }))
      expect(store.getState().nicknames).toEqual({
        b: 'two',
        [ME]: nickname,
      })
    })

    it('does not remove current user\'s nickanme', () => {
      store.dispatch(removeNickname({ userId: ME }))
      expect(store.getState().nicknames).toEqual({
        a: 'one',
        b: 'two',
        [ME]: nickname,
      })
    })

  })

})
