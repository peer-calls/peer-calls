package stats

import (
	"math/rand"
	"time"
)

// TODO Add function for computing the RTCP transmission interval according to RFC 3550
// Appendix A.7.

// 6.3 RTCP Packet Send and Receive Rules

//    The rules for how to send, and what to do when receiving an RTCP
//    packet are outlined here.  An implementation that allows operation in
//    a multicast environment or a multipoint unicast environment MUST meet
//    the requirements in Section 6.2.  Such an implementation MAY use the
//    algorithm defined in this section to meet those requirements, or MAY
//    use some other algorithm so long as it provides equivalent or better
//    performance.  An implementation which is constrained to two-party
//    unicast operation SHOULD still use randomization of the RTCP
//    transmission interval to avoid unintended synchronization of multiple
//    instances operating in the same environment, but MAY omit the "timer
//    reconsideration" and "reverse reconsideration" algorithms in Sections
//    6.3.3, 6.3.6 and 6.3.7.
//
//    To execute these rules, a session participant must maintain several
//    pieces of state:
//
//    tp: the last time an RTCP packet was transmitted;
//
//    tc: the current time;
//
//    tn: the next scheduled transmission time of an RTCP packet;
//
//    pmembers: the estimated number of session members at the time tn
//       was last recomputed;
//
//    members: the most current estimate for the number of session
//       members;
//
//    senders: the most current estimate for the number of senders in
//       the session;
//
//    rtcp_bw: The target RTCP bandwidth, i.e., the total bandwidth
//       that will be used for RTCP packets by all members of this session,
//       in octets per second.  This will be a specified fraction of the
//       "session bandwidth" parameter supplied to the application at
//       startup.
//
//    we_sent: Flag that is true if the application has sent data
//       since the 2nd previous RTCP report was transmitted.
//
//    avg_rtcp_size: The average compound RTCP packet size, in octets,
//       over all RTCP packets sent and received by this participant.  The
//       size includes lower-layer transport and network protocol headers
//       (e.g., UDP and IP) as explained in Section 6.2.
//
//    initial: Flag that is true if the application has not yet sent
//       an RTCP packet.
//
//    Many of these rules make use of the "calculated interval" between
//    packet transmissions.  This interval is described in the following
//    section.

// A.7 Computing the RTCP Transmission Interval
//
//    The following functions implement the RTCP transmission and reception
//    rules described in Section 6.2.  These rules are coded in several
//    functions:
//
//    o  rtcp_interval() computes the deterministic calculated interval,
//       measured in seconds.  The parameters are defined in Section 6.3.
//    o  OnExpire() is called when the RTCP transmission timer expires.
//
//    o  OnReceive() is called whenever an RTCP packet is received.
//
//    Both OnExpire() and OnReceive() have event e as an argument.  This is
//    the next scheduled event for that participant, either an RTCP report
//    or a BYE packet.  It is assumed that the following functions are
//    available:
//
//    o  Schedule(time t, event e) schedules an event e to occur at time t.
//       When time t arrives, the function OnExpire is called with e as an
//       argument.
//
//    o  Reschedule(time t, event e) reschedules a previously scheduled
//       event e for time t.
//
//    o  SendRTCPReport(event e) sends an RTCP report.
//
//    o  SendBYEPacket(event e) sends a BYE packet.
//
//    o  TypeOfEvent(event e) returns EVENT_BYE if the event being
//       processed is for a BYE packet to be sent, else it returns
//       EVENT_REPORT.
//
//    o  PacketType(p) returns PACKET_RTCP_REPORT if packet p is an RTCP
//       report (not BYE), PACKET_BYE if its a BYE RTCP packet, and
//       PACKET_RTP if its a regular RTP data packet.
//
//    o  ReceivedPacketSize() and SentPacketSize() return the size of the
//       referenced packet in octets.
//
//    o  NewMember(p) returns a 1 if the participant who sent packet p is
//       not currently in the member list, 0 otherwise.  Note this function
//       is not sufficient for a complete implementation because each CSRC
//       identifier in an RTP packet and each SSRC in a BYE packet should
//       be processed.
//
//    o  NewSender(p) returns a 1 if the participant who sent packet p is
//       not currently in the sender sublist of the member list, 0
//       otherwise.
//
//    o  AddMember() and RemoveMember() to add and remove participants from
//       the member list.
//
//    o  AddSender() and RemoveSender() to add and remove participants from
//       the sender sublist of the member list.
//
//    These functions would have to be extended for an implementation that
//    allows the RTCP bandwidth fractions for senders and non-senders to be
//    specified as explicit parameters rather than fixed values of 25% and
//    75%.  The extended implementation of rtcp_interval() would need to
//    avoid division by zero if one of the parameters was zero.
//
//    double rtcp_interval(int members,
//                         int senders,
//                         double rtcp_bw,
//                         int we_sent,
//                         double avg_rtcp_size,
//                         int initial)
//    {
//        /*
//         * Minimum average time between RTCP packets from this site (in
//         * seconds).  This time prevents the reports from `clumping' when
//         * sessions are small and the law of large numbers isn't helping
//         * to smooth out the traffic.  It also keeps the report interval
//         * from becoming ridiculously small during transient outages like
//         * a network partition.
//         */
//        double const RTCP_MIN_TIME = 5.;
//        /*
//         * Fraction of the RTCP bandwidth to be shared among active
//         * senders.  (This fraction was chosen so that in a typical
//         * session with one or two active senders, the computed report
//         * time would be roughly equal to the minimum report time so that
//         * we don't unnecessarily slow down receiver reports.)  The
//         * receiver fraction must be 1 - the sender fraction.
//         */
//        double const RTCP_SENDER_BW_FRACTION = 0.25;
//        double const RTCP_RCVR_BW_FRACTION = (1-RTCP_SENDER_BW_FRACTION);
//        /*
//        /* To compensate for "timer reconsideration" converging to a
//         * value below the intended average.
//         */
//        double const COMPENSATION = 2.71828 - 1.5;
//
//        double t;                   /* interval */
//        double rtcp_min_time = RTCP_MIN_TIME;
//        int n;                      /* no. of members for computation */
//
//        /*
//         * Very first call at application start-up uses half the min
//         * delay for quicker notification while still allowing some time
//         * before reporting for randomization and to learn about other
//         * sources so the report interval will converge to the correct
//         * interval more quickly.
//
//         */
//        if (initial) {
//            rtcp_min_time /= 2;
//        }
//        /*
//         * Dedicate a fraction of the RTCP bandwidth to senders unless
//         * the number of senders is large enough that their share is
//         * more than that fraction.
//         */
//        n = members;
//        if (senders <= members * RTCP_SENDER_BW_FRACTION) {
//            if (we_sent) {
//                rtcp_bw *= RTCP_SENDER_BW_FRACTION;
//                n = senders;
//            } else {
//                rtcp_bw *= RTCP_RCVR_BW_FRACTION;
//                n -= senders;
//            }
//        }
//
//        /*
//         * The effective number of sites times the average packet size is
//         * the total number of octets sent when each site sends a report.
//         * Dividing this by the effective bandwidth gives the time
//         * interval over which those packets must be sent in order to
//         * meet the bandwidth target, with a minimum enforced.  In that
//         * time interval we send one report so this time is also our
//         * average time between reports.
//         */
//        t = avg_rtcp_size * n / rtcp_bw;
//        if (t < rtcp_min_time) t = rtcp_min_time;
//
//        /*
//         * To avoid traffic bursts from unintended synchronization with
//         * other sites, we then pick our actual next report interval as a
//         * random number uniformly distributed between 0.5*t and 1.5*t.
//         */
//        t = t * (drand48() + 0.5);
//        t = t / COMPENSATION;
//        return t;
//    }
//
//    void OnExpire(event e,
//                  int    members,
//                  int    senders,
//                  double rtcp_bw,
//                  int    we_sent,
//                  double *avg_rtcp_size,
//                  int    *initial,
//                  time_tp   tc,
//                  time_tp   *tp,
//                  int    *pmembers)
//    {
//        /* This function is responsible for deciding whether to send an
//         * RTCP report or BYE packet now, or to reschedule transmission.
//         * It is also responsible for updating the pmembers, initial, tp,
//         * and avg_rtcp_size state variables.  This function should be
//         * called upon expiration of the event timer used by Schedule().
//         */
//
//        double t;     /* Interval */
//        double tn;    /* Next transmit time */
//
//        /* In the case of a BYE, we use "timer reconsideration" to
//         * reschedule the transmission of the BYE if necessary */
//
//        if (TypeOfEvent(e) == EVENT_BYE) {
//            t = rtcp_interval(members,
//                              senders,
//                              rtcp_bw,
//                              we_sent,
//                              *avg_rtcp_size,
//                              *initial);
//            tn = *tp + t;
//            if (tn <= tc) {
//                SendBYEPacket(e);
//                exit(1);
//            } else {
//                Schedule(tn, e);
//            }
//
//        } else if (TypeOfEvent(e) == EVENT_REPORT) {
//            t = rtcp_interval(members,
//                              senders,
//                              rtcp_bw,
//                              we_sent,
//                              *avg_rtcp_size,
//                              *initial);
//            tn = *tp + t;
//            if (tn <= tc) {
//                SendRTCPReport(e);
//                *avg_rtcp_size = (1./16.)*SentPacketSize(e) +
//                    (15./16.)*(*avg_rtcp_size);
//                *tp = tc;
//
//                /* We must redraw the interval.  Don't reuse the
//                   one computed above, since its not actually
//                   distributed the same, as we are conditioned
//                   on it being small enough to cause a packet to
//                   be sent */
//
//                t = rtcp_interval(members,
//                                  senders,
//                                  rtcp_bw,
//                                  we_sent,
//                                  *avg_rtcp_size,
//                                  *initial);
//
//                Schedule(t+tc,e);
//                *initial = 0;
//            } else {
//                Schedule(tn, e);
//            }
//            *pmembers = members;
//        }
//    }
//
//    void OnReceive(packet p,
//                   event e,
//                   int *members,
//                   int *pmembers,
//                   int *senders,
//                   double *avg_rtcp_size,
//                   double *tp,
//                   double tc,
//                   double tn)
//    {
//        /* What we do depends on whether we have left the group, and are
//         * waiting to send a BYE (TypeOfEvent(e) == EVENT_BYE) or an RTCP
//         * report.  p represents the packet that was just received.  */
//
//        if (PacketType(p) == PACKET_RTCP_REPORT) {
//            if (NewMember(p) && (TypeOfEvent(e) == EVENT_REPORT)) {
//                AddMember(p);
//                *members += 1;
//            }
//            *avg_rtcp_size = (1./16.)*ReceivedPacketSize(p) +
//                (15./16.)*(*avg_rtcp_size);
//        } else if (PacketType(p) == PACKET_RTP) {
//            if (NewMember(p) && (TypeOfEvent(e) == EVENT_REPORT)) {
//                AddMember(p);
//                *members += 1;
//            }
//            if (NewSender(p) && (TypeOfEvent(e) == EVENT_REPORT)) {
//                AddSender(p);
//                *senders += 1;
//            }
//        } else if (PacketType(p) == PACKET_BYE) {
//            *avg_rtcp_size = (1./16.)*ReceivedPacketSize(p) +
//                (15./16.)*(*avg_rtcp_size);
//
//            if (TypeOfEvent(e) == EVENT_REPORT) {
//                if (NewSender(p) == FALSE) {
//                    RemoveSender(p);
//                    *senders -= 1;
//                }
//
//                if (NewMember(p) == FALSE) {
//                    RemoveMember(p);
//                    *members -= 1;
//                }
//
//                if (*members < *pmembers) {
//                    tn = tc +
//                        (((double) *members)/(*pmembers))*(tn - tc);
//                    *tp = tc -
//                        (((double) *members)/(*pmembers))*(tc - *tp);
//
//                    /* Reschedule the next report for time tn */
//
//                    Reschedule(tn, e);
//                    *pmembers = *members;
//                }
//
//            } else if (TypeOfEvent(e) == EVENT_BYE) {
//                *members += 1;
//            }
//        }
//    }

type event int

const (
	eventBye = iota + 1
	eventReport
)

type scheduler struct {
	rand   *rand.Rand
	params *schedulerParams

	// the last time an RTCP packet was transmitted
	tp time.Time
	// the next scheduled time of an RTCP packet
	tn time.Time
	// the estimated number of session members at the last time tn was computed.
	pmembers int
	// the most current estimate for the number of session members
	members int
	// the most current estimate for the number of senders in the session.
	senders int
	// // The target RTCP bandwidth, i.e., the total bandwidth that will be used for
	// // RTCP packets by all members of this session, in octets per second.  This
	// // will be a specified fraction of the "session bandwidth" parameter supplied
	// // to the application at startup.
	// rtcp_bw float64
	// Flag that is true if the application has sent data since the 2nd previous
	// RTCP report was transmitted.
	we_sent bool
	// The average compound RTCP packet size, in octets,
	// over all RTCP packets sent and received by this participant.  The
	// size includes lower-layer transport and network protocol headers
	// (e.g., UDP and IP) as explained in Section 6.2.
	avg_rtcp_size float64
	// Flag that is true if the application has not yet sent an RTCP packet.
	initial bool
}

type schedulerParams struct {
	seed int64
	// The target RTCP bandwidth, i.e., the total bandwidth that will be used for
	// RTCP packets by all members of this session, in octets per second.  This
	// will be a specified fraction of the "session bandwidth" parameter supplied
	// to the application at startup.
	rtcp_bw float64
	// Minimum average time between RTCP packets from this site (in
	// seconds).  This time prevents the reports from `clumping' when
	// sessions are small and the law of large numbers isn't helping
	// to smooth out the traffic.  It also keeps the report interval
	// from becoming ridiculously small during transient outages like
	// a network partition.
	rtcpMinTime float64
	// Fraction of the RTCP bandwidth to be shared among active
	// senders.  (This fraction was chosen so that in a typical
	// session with one or two active senders, the computed report
	// time would be roughly equal to the minimum report time so that
	// we don't unnecessarily slow down receiver reports.)  The
	// receiver fraction must be 1 - the sender fraction.
	rtcpSenderBwFraction float64
	rtcpRcvrBwFraction   float64
	// const RTCP_RCVR_BW_FRACTION float64 = (1 - RTCP_SENDER_BW_FRACTION)
	// To compensate for "timer reconsideration" converging to a
	// value below the intended average.
	compensation float64
}

func (p *schedulerParams) defaults() {
	if p.rtcpMinTime == 0 {
		p.rtcpMinTime = 5.
	}

	if p.rtcpSenderBwFraction == 0 {
		p.rtcpSenderBwFraction = 0.25
	}

	if p.rtcpRcvrBwFraction == 0 {
		p.rtcpRcvrBwFraction = 1 - p.rtcpSenderBwFraction
	}

	if p.compensation == 0 {
		p.compensation = 2.71828 - 1.5
	}
}

func newScheduler(params schedulerParams) *scheduler {
	s := &scheduler{}

	params.defaults()

	s.params = &params
	s.rand = rand.New(rand.NewSource(params.seed))

	return s
}

func (s scheduler) rtcpInterval() time.Duration {
	// interval
	var (
		t             float64
		rtcp_min_time = s.params.rtcpMinTime
		// no. of members for computation
		n       int
		rtcp_bw float64
	)

	// Very first call at application start-up uses half the min
	// delay for quicker notification while still allowing some time
	// before reporting for randomization and to learn about other
	// sources so the report interval will converge to the correct
	// interval more quickly.
	if s.initial {
		rtcp_min_time /= 2
	}

	// Dedicate a fraction of the RTCP bandwidth to senders unless
	// the number of senders is large enough that their share is
	// more than that fraction.
	n = s.members
	if float64(s.senders) <= float64(s.members)*s.params.rtcpSenderBwFraction {
		if s.we_sent {
			rtcp_bw = s.params.rtcp_bw * s.params.rtcpSenderBwFraction
			n = s.senders
		} else {
			rtcp_bw = s.params.rtcp_bw * s.params.rtcpRcvrBwFraction
			n -= s.senders
		}
	}

	/*
	 * The effective number of sites times the average packet size is
	 * the total number of octets sent when each site sends a report.
	 * Dividing this by the effective bandwidth gives the time
	 * interval over which those packets must be sent in order to
	 * meet the bandwidth target, with a minimum enforced.  In that
	 * time interval we send one report so this time is also our
	 * average time between reports.
	 */
	t = s.avg_rtcp_size * float64(n) / rtcp_bw
	if t < rtcp_min_time {
		t = rtcp_min_time
	}

	/*
	 * To avoid traffic bursts from unintended synchronization with
	 * other sites, we then pick our actual next report interval as a
	 * random number uniformly distributed between 0.5*t and 1.5*t.
	 */
	t *= (s.rand.Float64() + 0.5)
	t /= s.params.compensation

	return time.Duration(t * float64(time.Second))
}

func (s *scheduler) SetLastRTCPPacketSize(lastRTCPPacketSize int) {
	s.avg_rtcp_size = (1./16.)*float64(lastRTCPPacketSize) + (15./16.)*(s.avg_rtcp_size)
}
