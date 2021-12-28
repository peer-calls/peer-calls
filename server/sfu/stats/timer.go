package stats

// import (
// 	"time"

// 	"github.com/peer-calls/peer-calls/v4/server/clock"
// )

// type Timer struct {
// 	params   *TimerParams
// 	c        chan RTCPReportRequest
// 	teardown chan struct{}
// 	torndown chan struct{}
// }

// type TimerParams struct {
// 	Clock     clock.Clock
// 	Scheduler *scheduler
// }

// type RTCPReportRequest struct {
// 	// Response contains the respones. The channel must always 1-buffered and the
// 	// recipient must always write a single response to the channel. The
// 	// recipient may or may not close the channel afterwards.
// 	Response chan RTCPReportResponse
// }

// type RTCPReportResponse struct {
// 	// PacketSize contains the bytes of the packet size
// 	PacketSize int
// 	// Err contains an error that occurred, if any.
// 	Err error
// }

// func NewTimer(params TimerParams) *Timer {
// 	tmr := &Timer{
// 		params:   &params,
// 		c:        make(chan RTCPReportRequest, 1),
// 		teardown: make(chan struct{}),
// 		torndown: make(chan struct{}),
// 	}

// 	go tmr.start()

// 	return tmr
// }

// func (tmr *Timer) C() <-chan RTCPReportRequest {
// 	return tmr.c
// }

// func (tmr *Timer) start() {
// 	defer func() {
// 		close(tmr.c)
// 		close(tmr.teardown)
// 	}()

// 	var (
// 		t = tmr.params.Scheduler.rtcpInterval()
// 		// tp is the last time an RTCP report was sent.
// 		tp    = time.Time{}
// 		timer = tmr.params.Clock.NewTimer(t)
// 	)

// 	defer timer.Stop()

// 	schedule := func(t time.Duration) {
// 		if !timer.Stop() {
// 			<-timer.C()
// 		}

// 		timer.Reset(t)
// 	}

// 	handleTick := func() {
// 		now := tmr.params.Clock.Now()

// 		t = tmr.params.Scheduler.rtcpInterval()

// 		// tn is the next transmission time.
// 		tn := tp.Add(t)

// 		if tn.After(now) {
// 			// next transmission time has changed in the meantime and is later than
// 			// it was.
// 			schedule(t)
// 		}

// 		req := RTCPReportRequest{
// 			Response: make(chan RTCPReportResponse, 1),
// 		}

// 		var res RTCPReportResponse

// 		select {
// 		case tmr.c <- req:
// 		case <-tmr.teardown:
// 			return
// 		}

// 		select {
// 		case res = <-req.Response:
// 		case <-tmr.teardown:
// 			return
// 		}

// 		tmr.params.Scheduler.SetLastRTCPPacketSize(res.PacketSize)

// 		tp = tmr.params.Clock.Now()
// 		t = tmr.params.Scheduler.rtcpInterval()

// 		schedule(t)
// 	}

// 	for {
// 		select {
// 		case <-timer.C():
// 			handleTick()
// 		case <-tmr.teardown:
// 			return
// 		}
// 	}
// }

// func (tmr *Timer) Close() {
// 	close(tmr.teardown)
// 	<-tmr.torndown
// }
