# Server Transport

## SCTP Association reconnect

2020-12-15T09:21:05.687877+01:00 [pion:sctp:error] Failed to handle chunk: todo: handle Init when in state Established

The reason for the above error was the fact that sctp.Association.Close call
does not do shutdown.

Proposed workaround:

0. Use SCTP on the factory level instead on the metadata/data transport level.
   RTP/RTCP is still sent via UDP (done)

1. Add a ticker that sends an event over a separate sctp channel ever N
   seconds.
2. Add a ticker that verifies there was a tick event in the last M*N seconds.
3. Reconnect factories if (2) fails.

## Transport Add/Removal States

1. Creating a room on node A (peer joins) and adding a track triggers creation
   of server transport on node B.
2. Node B might not have any peers just yet - there should be no need to for
   this transport to exist yet. FIXME
3. When a peer joins to same room on node B and disconnects, the server
   transport on node B will be closed.
4. When another peer joins to the same roon on node B again, the server
   transport on node B will be created again. With the new version of
   udptransport, we have an init event that will sync the tracks.

   However, this init event now triggers weird situation on short-lived
   connections, as the flakey test in udptransport:

   ```console
   PEERCALLS_LOG="**:metadata_transport:trace" go test ./server/udptransport2/ --count=1
   ```

   Node1 ServerTransportA connects, sends init
   Node1 ServerTransportA sends track events

   Node2 ServerTransportA is created, sends init
   Node2 ServerTransportA receives track event
   Node2 ServerTransportA closes

   Node2 ServerTransportA is recreated after receiving a track event from init

### Question?

Could there be a way to use a new control channel in udptransport2.Factory to
handshake the creations of room transports?

E.g.:

1. Node A room X recieves a peer.
2. Node A lets its factory know there is a peer in room X
3. Node B - nothing yet
4. Node B receives a peer in room X
5. Node B notifies its factory there is a peer in room X
6. Node B factory creates a transport for room X
7. Node A's factory creates a transport for room X

There could still be an issue with synchronizing short-lived node-to-node
connections. For example:

1. Node A room X recieves a peer.
2. Node A lets its factory know there is a peer in room X
3. Node B - nothing yet
4. Node B receives a peer in room X
5. Around the same time, the peer in room X on node A just left, so room is destroyed.
6. Node B notifies its factory there is a peer in room X
7. Node B factory creates a transport for room X
8. Node A realizes it no longer has a peer in room X.
9. Node B receives the info that the peer just left so disconnects.

Perhaps udpmux/stringmux connections shouldn't be created automatically as soon
as the first packet is received.
