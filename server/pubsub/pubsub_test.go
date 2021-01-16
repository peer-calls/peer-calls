package pubsub_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/pubsub"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestPubSub(t *testing.T) {
	defer goleak.VerifyNone(t)

	ps := pubsub.New(logger.New())

	defer ps.Close()

	t1 := newTransportMock("a")
	t2 := newTransportMock("b")
	t3 := newTransportMock("c")

	_ = t2

	type track struct {
		clientID string
		ssrc     uint32
	}

	type pub struct {
		clientID string
		track    transport.Track
	}

	type sub struct {
		clientID  string
		ssrc      uint32
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
		wantTracks map[*transportMock]map[uint32]transport.Track
	}

	testCases := []testCase{
		{
			descr: "subscribe, get error because track does not exist",
			sub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      1,
				transport: t3,
			},
			wantErr: pubsub.ErrTrackNotFound,
			wantSubs: map[track][]string{
				{"a", 1}: nil,
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "publish first track, still no subscribers",
			pub: &pub{
				clientID: t1.ClientID(),
				track:    transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
			},
			wantErr: nil,
			wantSubs: map[track][]string{
				{"a", 1}: nil,
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "subscribe to own track, error",
			sub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      1,
				transport: t1,
			},
			wantErr: pubsub.ErrSubscribeToOwnTrack,
			wantSubs: map[track][]string{
				{"a", 1}: nil,
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "subscribe to track, success",
			sub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      1,
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
				t3: {},
			},
		},
		{
			descr: "subscribe to track (again), error",
			sub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      1,
				transport: t2,
			},
			wantErr: errTrackAlreadyAdded,
			wantSubs: map[track][]string{
				{"a", 1}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
				t3: {},
			},
		},
		{
			descr: "subscribe to track from another transport, success",
			sub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      1,
				transport: t3,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t2.ClientID(), t3.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t2: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "unsubscribe from non existing track, error",
			unsub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      2,
				transport: t3,
			},
			wantErr: pubsub.ErrTrackNotFound,
			wantSubs: map[track][]string{
				{"a", 1}: {t2.ClientID(), t3.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "unsubscribe from subscribed track, success",
			unsub: &sub{
				clientID:  t1.ClientID(),
				ssrc:      1,
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {},
				t2: {},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "publish track from t3",
			pub: &pub{
				clientID: t3.ClientID(),
				track:    transport.NewSimpleTrack("t3", 8, 3, "C", "CC"),
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"c", 3}: nil,
			},
		},
		{
			descr: "publish another track from t3",
			pub: &pub{
				clientID: t3.ClientID(),
				track:    transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"c", 3}: nil,
			},
		},
		{
			descr: "subscribe to track 3, success",
			sub: &sub{
				clientID:  t3.ClientID(),
				ssrc:      3,
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"c", 3}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t2: {
					3: transport.NewSimpleTrack("t3", 8, 3, "C", "CC"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "subscribe to track 4, success",
			sub: &sub{
				clientID:  t3.ClientID(),
				ssrc:      4,
				transport: t2,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"c", 3}: {t2.ClientID()},
				{"c", 4}: {t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t2: {
					3: transport.NewSimpleTrack("t3", 8, 3, "C", "CC"),
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "subscribe to track 4 from t1, success",
			sub: &sub{
				clientID:  t3.ClientID(),
				ssrc:      4,
				transport: t1,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"c", 3}: {t2.ClientID()},
				{"c", 4}: {t1.ClientID(), t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
				},
				t2: {
					3: transport.NewSimpleTrack("t3", 8, 3, "C", "CC"),
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "pub track 5 from t2, success",
			pub: &pub{
				clientID: t2.ClientID(),
				track:    transport.NewSimpleTrack("t2", 8, 5, "E", "EE"),
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"c", 3}: {t2.ClientID()},
				{"c", 4}: {t1.ClientID(), t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
				},
				t2: {
					3: transport.NewSimpleTrack("t3", 8, 3, "C", "CC"),
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr: "sub to track 5 from t1, success",
			sub: &sub{
				clientID:  t2.ClientID(),
				ssrc:      5,
				transport: t1,
			},
			wantSubs: map[track][]string{
				{"a", 1}: {t3.ClientID()},
				{"b", 5}: {t1.ClientID()},
				{"c", 3}: {t2.ClientID()},
				{"c", 4}: {t1.ClientID(), t2.ClientID()},
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
					5: transport.NewSimpleTrack("t2", 8, 5, "E", "EE"),
				},
				t2: {
					3: transport.NewSimpleTrack("t3", 8, 3, "C", "CC"),
					4: transport.NewSimpleTrack("t3", 8, 4, "D", "DD"),
				},
				t3: {
					1: transport.NewSimpleTrack("t1", 8, 1, "A", "AA"),
				},
			},
		},
		{
			descr:     "terminate t3, unpublish and unsubscribe, but keep other tracks",
			terminate: t3.ClientID(),
			wantSubs: map[track][]string{
				{"a", 1}: nil,
				{"b", 5}: {t1.ClientID()},
				{"c", 3}: nil,
				{"c", 4}: nil,
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
				t1: {
					5: transport.NewSimpleTrack("t2", 8, 5, "E", "EE"),
				},
				t2: {},
				t3: {},
			},
		},
		{
			descr: "unpub track 5, unsubscribe from t2",
			unpub: &track{
				clientID: t2.ClientID(),
				ssrc:     5,
			},
			wantSubs: map[track][]string{
				{"a", 1}: nil,
				{"b", 5}: nil,
				{"c", 3}: nil,
				{"c", 4}: nil,
			},
			wantTracks: map[*transportMock]map[uint32]transport.Track{
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
			ps.Pub(tc.pub.clientID, tc.pub.track)
		case tc.unpub != nil:
			ps.Unpub(tc.unpub.clientID, tc.unpub.ssrc)
		case tc.sub != nil:
			err = ps.Sub(tc.sub.clientID, tc.sub.ssrc, tc.sub.transport)
		case tc.unsub != nil:
			err = ps.Unsub(tc.unsub.clientID, tc.unsub.ssrc, tc.unsub.transport.ClientID())
		case tc.terminate != "":
			ps.Terminate(tc.terminate)
		}

		assert.Equal(t, tc.wantErr, errors.Cause(err), "wantErr: %s", descr)

		var gotSubs map[track][]string

		if tc.wantSubs != nil {
			gotSubs = map[track][]string{}

			for k := range tc.wantSubs {
				gotSubs[k] = []string(nil)

				for _, subscriber := range ps.Subscribers(k.clientID, k.ssrc) {
					gotSubs[k] = append(gotSubs[k], subscriber.ClientID())
				}

				sort.Strings(gotSubs[k])
			}
		}

		assert.Equal(t, tc.wantSubs, gotSubs, "wantSubs: %s", descr)

		gotTracks := map[string]map[uint32]transport.Track{}
		wantTracks := map[string]map[uint32]transport.Track{}

		for k, v := range tc.wantTracks {
			gotTracks[k.ClientID()] = k.addedTracks
			wantTracks[k.ClientID()] = v
		}

		assert.Equal(t, wantTracks, gotTracks, "wantTracks: %s", descr)
	}

	ps.Subscribers("clientID", 9)
}

type transportMock struct {
	clientID    string
	addedTracks map[uint32]transport.Track
}

func newTransportMock(clientID string) *transportMock {
	return &transportMock{
		clientID:    clientID,
		addedTracks: map[uint32]transport.Track{},
	}
}

func (t *transportMock) ClientID() string {
	return t.clientID
}

var (
	errTrackAlreadyAdded = errors.Errorf("track already added")
	errTrackNotFound     = errors.Errorf("track not found")
)

func (t *transportMock) AddTrack(track transport.Track) error {
	if _, ok := t.addedTracks[track.SSRC()]; ok {
		return errors.Annotatef(errTrackAlreadyAdded, "%d", track.SSRC())
	}

	t.addedTracks[track.SSRC()] = track

	return nil
}

func (t *transportMock) RemoveTrack(ssrc uint32) error {
	if _, ok := t.addedTracks[ssrc]; !ok {
		return errors.Annotatef(errTrackNotFound, "%d", ssrc)
	}

	delete(t.addedTracks, ssrc)

	return nil
}

var _ pubsub.Transport = &transportMock{}
