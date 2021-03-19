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
2. Add a ticker that verifies there was a tick event in the last `M*N` seconds.
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

I believe this is addresesd by the control_state_tracker, we'd only need to
reimplement NodeManager to make use of the ControlTranpsort after a Factory
is created. Or actually factory can be reimplemented to accept only two types
of requests: WantCreate wand WantClose, which is called every time a room is
created and removed, respectively.

Consequently, the factory should have the internal state tracker and figure out
when to create transports and send them to transports channel. NewTransport
method will be replaced with Create(roomID string) and a new method will be
added, called Close(roomID string). These methods will initate the handshake
via the control transport and once the transport is deemed to be ready and
added to the room, it will be added by the node manager.

forget this: ~One caveat: currently the rooms are closed when the last peer
leaves, that means the room transport events would be left unhandled.~

Transports created because a packet was received should be destroyed - and it
should be considered a bug if that happens.

TODO Figure out when a transport factory is dead (e.g. a connection broke and
we need to recreate the factory. This can be implemented with pings in the
control channel, and could actually be done internally in factory.  I don't
think this is an immediate problem that needs to be handled right now, but
something to think about in the future.

# Sender and Receiver reports

Sender and receiver reports should be taken into account, and the SFU should
produce these reports.

For example, imagine a sfu with 3 peers in the same room, each is sending a
video and an audio track. Peer A is interested in both tracks from other peers,
while peer B is only intersted in audio tracks and peer C only in video tracks.


+--------+   SSRC 1 audio           +-------------+
| Peer A |------------------------->| SFU peer A' |>--------------+
|        |                          |             |               |
|        |   SSRC 2 video           |             |               |
|        |------------------------->|             |>--------------|--+
|        |                          |             |               |  |
|        |   SSRC 3 audio           |             |               |  |
|        |<-------------------------|             |<-+            |  |
|        |                          |             |  |            |  |
|        |   SSRC 4 video           |             |  |            |  |
|        |<-------------------------|             |<-|--+         |  |
|        |                          |             |  |  |         |  |
|        |   SSRC 5 video           |             |  |  |         |  |
|        |<-------------------------|             |<-|--|--+      |  |
|        |                          |             |  |  |  |      |  |
|        |   SSRC 6 video           |             |  |  |  |      |  |
|        |<-------------------------|             |<-|--|--|--+   |  |
+--------+                          +-------------+  |  |  |  |   |  |
                                                     |  |  |  |   |  |
+--------+   SSRC 3 audio           +-------------+  |  |  |  |   |  |
| Peer B |------------------------->| SFU peer B' |>-+  |  |  |   |  |
|        |                          |             |     |  |  |   |  |
|        |   SSRC 4 video           |             |     |  |  |   |  |
|        |------------------------->|             |>----+  |  |   |  |
|        |                          |             |     |  |  |   |  |
|        |   SSRC 1 audio           |             |     |  |  |   |  |
|        |<-------------------------|             |<----|--|--|---+  |
|        |                          |             |     |  |  |      |
|        |   SSRC 5 video           |             |     |  |  |      |
|        |<-------------------------|             |<----|--+  |      |
+--------+                          +-------------+     |  |  |      |
                                                        |  |  |      |
+--------+   SSRC 5 audio           +-------------+     |  |  |      |
| Peer C |------------------------->| SFU peer C' |>----|--+  |      |
|        |                          |             |     |     |      |
|        |   SSRC 6 video           |             |     |     |      |
|        |------------------------->|             |>----|-----+      |
|        |                          |             |     |            |
|        |   SSRC 2 video           |             |     |            |
|        |<-------------------------|             |<----|------------+
|        |                          |             |     |
|        |   SSRC 4 video           |             |     |
|        |<-------------------------|             |<----+
+--------+                          +-------------+

In the scenario above:

- Peer A should send sender reports for SSRCs 1 and 2, and reception reports
  for SSRCs 3, 4, 5, and 6.
- Peer B should send sender reports for SSRCs 3 and 4, and reception reports
  for SSRCs 1 and 5.
- Peer C should send sender reports for SSRCs 5 and 6, and reception reports
  for SSRCs 2 and 4.

In the screen above, the peers are usually web browsers and they already send
this information.

On the SFU side,

Peer A' should:

1. Send reports to peer A: sender reports for SSRCs 3, 4, 5, and 6 to Peer
   A, and reception reports for SSRCs 1 and 2.

Peer B' should:

1. Send reports to peer B: sender reports for SSRCs 1 and 5, and reception
   reports for SSRcs 3 and 4.

Peer C' should:

1. Send reports to peer C: sender reports for SSRCs 2 and 4, and reception
   reports for SSRcs 5 and 6.


One might think the following: It does not make sense to send any kinds of
reports between SFU peers because they are already on the same node. This is
true, however not completely. There is no reason for peers A', B' and C' to
send RTCP packets to each other, but the SFU should send

But,
this is not true. Peer B should be able to adjust the send rate[1] if the SFU
notices that packets for SSRC 4 aren't being delivered on time to its
subcsribers* (to Peer A and C). Instead, the reception reports for SSRC 4
should be forwarded to the originating node (SFU peer B'), which should decide
which stats to use before sending a report to Peer B'.

When there are no track subscribers on the SFU, the SFU might just calculate
the reception reports on its own. But when subscribers exist, the reception
reports from these subscribers should be used (combined).

[1]: *Of course, it wouldn't make sense to lower the bitrate for all peers that
are capable of receiving all packets on time, and there's a single peer with a
bad connection. More on this later (this can probably be solved by Simulcast,
at least partially).
