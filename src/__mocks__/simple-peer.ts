/* eslint-disable */
import { EventEmitter } from 'events'

const Peer = jest.fn().mockImplementation(() => {
  const peer = new EventEmitter();
  (peer as any).destroy = jest.fn();
  (peer as any).signal = jest.fn();
  (peer as any).send = jest.fn();
  (peer as any).addTrack = jest.fn();
  (peer as any).removeTrack = jest.fn();
  (peer as any).addStream = jest.fn();
  (peer as any).removeStream = jest.fn();
  (Peer as any).instances.push(peer)
  return peer
});

(Peer as any).instances = []
export default Peer
