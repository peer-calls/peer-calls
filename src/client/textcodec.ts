import { TextDecoder as TextDecoderPolyfill, TextEncoder as TextEncoderPolyfill } from 'fastestsmallesttextencoderdecoder'

export const TextEncoder = window.TextEncoder || TextEncoderPolyfill
export const TextDecoder = window.TextDecoder || TextDecoderPolyfill
