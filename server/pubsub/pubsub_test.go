package pubsub_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pubsub"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestPubSub(t *testing.T) {
	defer goleak.VerifyNone(t)

	ps := pubsub.New(logger.NewFromEnv("LOG"))

	defer ps.Close()

	t1 := newTransportMock("a")
	t2 := newTransportMock("b")
	t3 := newTransportMock("c")

	_ = t2

	type track struct {
		clientID string
		trackID  transport.TrackID
	}

	type pub struct {
		clientID string
		track    transport.Track
	}

	type sub struct {
		clientID  string
		trackID   transport.TrackID
		transport pubsub.Transport
	}

	type testCase struct {
		descr string

		pub       *pub
		unpub     *track
		sub       *sub
		unsub     *sub
		terminate string

		wantErr    error
		wantSubs   map[track][]string
		wantTracks map[*transportMock]map[transport.TrackID]transport.Track
	}

	testCases := []testCase{
		{
			descr: "subscribe, get error because track does not exist",
			sub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track1",
				transport: t3,
			},
			wantErr: pubsub.ErrTrackNotFound,
			wantSubs: map[track][]string{
				{"a", "A:track1"}: nil,
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "publish first track, still no subscribers",
			pub: &pub{
				clientID: t1.ClientID(),
				track:    transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
			},
			wantErr: nil,
			wantSubs: map[track][]string{
				{"a", "A:track1"}: nil,
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "subscribe to own track, error",
			sub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track1",
				transport: t1,
			},
			wantErr: pubsub.ErrSubscribeToOwnTrack,
			wantSubs: map[track][]string{
				{"a", "A:track1"}: nil,
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "subscribe to track, success",
			sub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track1",
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
				t3: {},
			},
		},
		{
			descr: "subscribe to track (again), error",
			sub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track1",
				transport: t2,
			},
			wantErr: errTrackAlreadyAdded,
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
				t3: {},
			},
		},
		{
			descr: "subscribe to track from another transport, success",
			sub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track1",
				transport: t3,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t2.ClientID(), t3.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t2: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "unsubscribe from non existing track, error",
			unsub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track2",
				transport: t3,
			},
			wantErr: pubsub.ErrTrackNotFound,
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t2.ClientID(), t3.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "unsubscribe from subscribed track, success",
			unsub: &sub{
				clientID:  t1.ClientID(),
				trackID:   "A:track1",
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "publish track from t3",
			pub: &pub{
				clientID: t3.ClientID(),
				track:    transport.NewSimpleTrack("track3", "C", "audio/opus", "CC"),
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"c", "C:track3"}: nil,
			},
		},
		{
			descr: "publish another track from t3",
			pub: &pub{
				clientID: t3.ClientID(),
				track:    transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"c", "C:track3"}: nil,
			},
		},
		{
			descr: "subscribe to track 3, success",
			sub: &sub{
				clientID:  t3.ClientID(),
				trackID:   "C:track3",
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"c", "C:track3"}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t2: {
					"C:track3": transport.NewSimpleTrack("track3", "C", "audio/opus", "CC"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "subscribe to track 4, success",
			sub: &sub{
				clientID:  t3.ClientID(),
				trackID:   "D:track4",
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"c", "C:track3"}: {t2.ClientID()},
				{"c", "D:track4"}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t2: {
					"C:track3": transport.NewSimpleTrack("track3", "C", "audio/opus", "CC"),
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "subscribe to track 4 from t1, success",
			sub: &sub{
				clientID:  t3.ClientID(),
				trackID:   "D:track4",
				transport: t1,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"c", "C:track3"}: {t2.ClientID()},
				{"c", "D:track4"}: {t1.ClientID(), t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
				},
				t2: {
					"C:track3": transport.NewSimpleTrack("track3", "C", "audio/opus", "CC"),
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "pub track 5 from t2, success",
			pub: &pub{
				clientID: t2.ClientID(),
				track:    transport.NewSimpleTrack("track5", "E", "audio/opus", "EE"),
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"c", "C:track3"}: {t2.ClientID()},
				{"c", "D:track4"}: {t1.ClientID(), t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
				},
				t2: {
					"C:track3": transport.NewSimpleTrack("track3", "C", "audio/opus", "CC"),
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr: "sub to track 5 from t1, success",
			sub: &sub{
				clientID:  t2.ClientID(),
				trackID:   "E:track5",
				transport: t1,
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: {t3.ClientID()},
				{"b", "E:track5"}: {t1.ClientID()},
				{"c", "C:track3"}: {t2.ClientID()},
				{"c", "D:track4"}: {t1.ClientID(), t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
					"E:track5": transport.NewSimpleTrack("track5", "E", "audio/opus", "EE"),
				},
				t2: {
					"C:track3": transport.NewSimpleTrack("track3", "C", "audio/opus", "CC"),
					"D:track4": transport.NewSimpleTrack("track4", "D", "audio/opus", "DD"),
				},
				t3: {
					"A:track1": transport.NewSimpleTrack("track1", "A", "audio/opus", "AA"),
				},
			},
		},
		{
			descr:     "terminate t3, unpublish and unsubscribe, but keep other tracks",
			terminate: t3.ClientID(),
			wantSubs: map[track][]string{
				{"a", "A:track1"}: nil,
				{"b", "E:track5"}: {t1.ClientID()},
				{"c", "C:track3"}: nil,
				{"c", "D:track4"}: nil,
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {
					"E:track5": transport.NewSimpleTrack("track5", "E", "audio/opus", "EE"),
				},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "unpub track 5, unsubscribe from t2",
			unpub: &track{
				clientID: t2.ClientID(),
				trackID:  "E:track5",
			},
			wantSubs: map[track][]string{
				{"a", "A:track1"}: nil,
				{"b", "E:track5"}: nil,
				{"c", "C:track3"}: nil,
				{"c", "D:track4"}: nil,
			},
			wantTracks: map[*transportMock]map[transport.TrackID]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
	}

	for i, tc := range testCases {
		descr := fmt.Sprintf("%d. %s", i, tc.descr)

		var err error

		switch {
		case tc.pub != nil:
			ps.Pub(tc.pub.clientID, newReaderMock(tc.pub.track))
		case tc.unpub != nil:
			ps.Unpub(tc.unpub.clientID, tc.unpub.trackID)
		case tc.sub != nil:
			_, err = ps.Sub(tc.sub.clientID, tc.sub.trackID, tc.sub.transport)
		case tc.unsub != nil:
			err = ps.Unsub(tc.unsub.clientID, tc.unsub.trackID, tc.unsub.transport.ClientID())
		case tc.terminate != "":
			ps.Terminate(tc.terminate)
		}

		assert.Equal(t, tc.wantErr, errors.Cause(err), "wantErr: %s", descr)

		var gotSubs map[track][]string

		if tc.wantSubs != nil {
			gotSubs = map[track][]string{}

			for k := range tc.wantSubs {
				gotSubs[k] = []string(nil)

				gotSubs[k] = append(gotSubs[k], ps.Subscribers(k.clientID, k.trackID)...)

				sort.Strings(gotSubs[k])
			}
		}

		assert.Equal(t, tc.wantSubs, gotSubs, "wantSubs: %s", descr)

		gotTracks := map[string]map[transport.TrackID]transport.Track{}
		wantTracks := map[string]map[transport.TrackID]transport.Track{}

		for k, v := range tc.wantTracks {
			gotTracks[k.ClientID()] = k.addedTracks
			wantTracks[k.ClientID()] = v
		}

		assert.Equal(t, wantTracks, gotTracks, "wantTracks: %s", descr)
	}
}

type transportMock struct {
	clientID    string
	addedTracks map[transport.TrackID]transport.Track
}

func newTransportMock(clientID string) *transportMock {
	return &transportMock{
		clientID:    clientID,
		addedTracks: map[transport.TrackID]transport.Track{},
	}
}

func (t *transportMock) ClientID() string {
	return t.clientID
}

var (
	errTrackAlreadyAdded = errors.Errorf("track already added")
	errTrackNotFound     = errors.Errorf("track not found")
)

func (t *transportMock) AddTrack(track transport.Track) (transport.TrackLocal, transport.Sender, error) {
	if _, ok := t.addedTracks[track.UniqueID()]; ok {
		return nil, nil, errors.Annotatef(errTrackAlreadyAdded, "%s", track.UniqueID())
	}

	t.addedTracks[track.UniqueID()] = track

	return trackLocalMock{t.clientID, track}, senderMock{}, nil
}

func (t *transportMock) RemoveTrack(trackID transport.TrackID) error {
	if _, ok := t.addedTracks[trackID]; !ok {
		return errors.Annotatef(errTrackNotFound, "%s", trackID)
	}

	delete(t.addedTracks, trackID)

	return nil
}

var _ pubsub.Transport = &transportMock{}

type senderMock struct {
	transport.Sender
}

type trackLocalMock struct {
	clientID string
	track    transport.Track
}

func (t trackLocalMock) Track() transport.Track {
	return t.track
}

func (t trackLocalMock) Write(b []byte) (int, error) {
	return 0, nil
}

func (t trackLocalMock) WriteRTP(b *rtp.Packet) error {
	return nil
}

var _ transport.TrackLocal = trackLocalMock{}

type readerMock struct {
	track transport.Track
	subs  map[string]transport.Track
}

func newReaderMock(track transport.Track) *readerMock {
	return &readerMock{
		track: track,
		subs:  map[string]transport.Track{},
	}
}

func (r *readerMock) Track() transport.Track {
	return r.track
}

func (r *readerMock) Sub(subClientID string, trackLocal transport.TrackLocal) error {
	if _, ok := r.subs[subClientID]; ok {
		return errors.Errorf("client is already subscribed to track: %s: %+v", subClientID, trackLocal.Track())
	}

	r.subs[subClientID] = trackLocal.Track()

	return nil
}

func (r *readerMock) Unsub(subClientID string) error {
	if _, ok := r.subs[subClientID]; !ok {
		return errors.Errorf("client sub not found: %s: %+v", subClientID, r.track)
	}

	delete(r.subs, subClientID)

	return nil
}

func (r *readerMock) Subs() []string {
	subs := make([]string, len(r.subs))

	i := -1
	for k := range r.subs {
		i++
		subs[i] = k
	}

	return subs
}

func (r *readerMock) SSRC() webrtc.SSRC {
	return webrtc.SSRC(0)
}

func (r *readerMock) RID() string {
	return ""
}

var _ pubsub.Reader = &readerMock{}
