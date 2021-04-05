package cli

import (
	"context"
	"fmt"
	"net"
	"sync"

	"strconv"

	"net/url"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/command"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/spf13/pflag"
)

type playHandler struct {
	args struct {
		streams []string
		// ffmpeg  string
	}

	log logger.Logger
	wg  sync.WaitGroup
}

func (h *playHandler) RegisterFlags(c *command.Command, flags *pflag.FlagSet) {
	flags.StringArrayVarP(&h.args.streams, "streams", "s", nil, "streams to read RTP from")
}

func (h *playHandler) listenUDP(host string, port int) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(host),
		Port: port,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return conn, nil
}

func (h *playHandler) isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (h *playHandler) startRTPLoop(ctx context.Context, streamURL *url.URL) error {
	q := streamURL.Query()

	pktSize, _ := strconv.Atoi(q.Get("pkt_size"))
	if pktSize == 0 {
		pktSize = 1200
	}

	port, err := strconv.Atoi(streamURL.Port())
	if err != nil {
		return errors.Trace(err)
	}

	conn, err := h.listenUDP(streamURL.Hostname(), port)
	if err != nil {
		return errors.Trace(err)
	}

	h.wg.Add(1)

	go func() {
		defer h.wg.Done()

		<-ctx.Done()
		conn.Close()
	}()

	buf := make([]byte, pktSize)

	for !h.isDone(ctx) {
		i, _, err := conn.ReadFrom(buf)
		if err != nil {
			h.log.Error("read err", err, nil)

			continue
		}

		var pkt rtp.Packet

		if err := pkt.Unmarshal(buf[:i]); err != nil {
			h.log.Error("unmarshal RTP", err, nil)

			continue
		}

		fmt.Println(pkt)

		// TODO write to peer connection.
	}

	return nil
}

func (h *playHandler) startRTCPLoop(ctx context.Context, streamURL *url.URL) error {
	q := streamURL.Query()

	pktSize, _ := strconv.Atoi(q.Get("pkt_size"))
	if pktSize == 0 {
		pktSize = 1200
	}

	rtpPort, err := strconv.Atoi(streamURL.Port())
	if err != nil {
		return errors.Trace(err)
	}

	port, _ := strconv.Atoi(q.Get("rtcpport"))
	if port == 0 {
		port = rtpPort + 1
	}

	conn, err := h.listenUDP(streamURL.Hostname(), port)
	if err != nil {
		return errors.Trace(err)
	}

	h.wg.Add(1)

	go func() {
		defer h.wg.Done()

		<-ctx.Done()
		conn.Close()
	}()

	buf := make([]byte, pktSize)

	for !h.isDone(ctx) {
		i, _, err := conn.ReadFrom(buf)
		if err != nil {
			h.log.Error("read err", err, nil)

			continue
		}

		pkt, err := rtcp.Unmarshal(buf[:i])
		if err != nil {
			h.log.Error("unmarshal RTCP", err, nil)

			continue
		}

		fmt.Println(pkt)

		// TODO write to sender.
	}

	return nil
}

func (h *playHandler) Handle(ctx context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var errs multierr.Sync

	for _, stream := range h.args.streams {
		streamURL, err := url.Parse(stream)
		if err != nil {
			errs.Add(errors.Annotatef(err, "stream: %s", streamURL))
			cancel()

			continue
		}

		if streamURL.Scheme != "rtp" {
			errs.Add(errors.Errorf("only rtp:// is supported, but got: %s", stream))
			cancel()

			continue
		}

		h.wg.Add(1)

		go func() {
			defer h.wg.Done()

			if err := h.startRTPLoop(ctx, streamURL); err != nil {
				errs.Add(errors.Annotate(err, "read RTP"))
				cancel()
			}
		}()

		h.wg.Add(1)

		go func() {
			defer h.wg.Done()

			if err := h.startRTCPLoop(ctx, streamURL); err != nil {
				errs.Add(errors.Annotate(err, "read RTCP"))
				cancel()
			}
		}()
	}

	h.wg.Wait()

	return errors.Trace(errs.Err())
}

func newPlayCmd(props Props) *command.Command {
	h := &playHandler{
		log: props.Log,
	}

	return command.New(command.Params{
		Name:         "play",
		Desc:         "Play RTP streams",
		FlagRegistry: h,
		Handler:      h,
	})
}
