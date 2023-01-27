package codecs

import (
	"strings"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

var ErrUnsupportedMimeType = errors.Errorf("unsupported mime type")

type Registry struct {
	Audio Props
	Video Props
}

type Props struct {
	CodecParameters  []webrtc.RTPCodecParameters
	HeaderExtensions []HeaderExtension
}

type HeaderExtension struct {
	Parameter         webrtc.RTPHeaderExtensionParameter
	AllowedDirections []webrtc.RTPTransceiverDirection
}

const (
	clockRateOpus   = 48000
	PayloadTypeOpus = 111
	channelsOpus    = 2
)

func opus() webrtc.RTPCodecCapability {
	return webrtc.RTPCodecCapability{
		MimeType:     webrtc.MimeTypeOpus,
		ClockRate:    clockRateOpus,
		Channels:     channelsOpus,
		SDPFmtpLine:  "minptime=10;useinbandfec=1",
		RTCPFeedback: nil,
	}
}

func NewRegistryDefault() *Registry {
	videoRTCPFeedback := []webrtc.RTCPFeedback{
		{
			Type:      "goog-remb",
			Parameter: "",
		},
		{
			Type:      "ccm",
			Parameter: "fir",
		},
		{
			Type:      "nack",
			Parameter: "",
		},
		{
			Type:      "nack",
			Parameter: "pli",
		},
	}

	return &Registry{
		Audio: Props{
			CodecParameters: []webrtc.RTPCodecParameters{
				{
					RTPCodecCapability: opus(),
					PayloadType:        PayloadTypeOpus,
				},
			},
			HeaderExtensions: nil,
		},
		Video: Props{
			CodecParameters: []webrtc.RTPCodecParameters{
				// {
				// 	RTPCodecCapability: webrtc.RTPCodecCapability{MimeTypeVP9, 90000, 0, "profile-id=0", videoRTCPFeedback},
				// 	PayloadType:        98,
				// },
				// {
				// 	RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=98", nil},
				// 	PayloadType:        99,
				// },
				// {
				// 	RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeVP9, 90000, 0, "profile-id=1", videoRTCPFeedback},
				// 	PayloadType:        100,
				// },
				// {
				// 	RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=100", nil},
				// 	PayloadType:        101,
				// },

				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeH264,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 102,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=102",
						RTCPFeedback: nil,
					},
					PayloadType: 121,
				},

				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeH264,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 127,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=127",
						RTCPFeedback: nil,
					},
					PayloadType: 120,
				},

				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeH264,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 125,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=125",
						RTCPFeedback: nil,
					},
					PayloadType: 107,
				},

				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeH264,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 108,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=108",
						RTCPFeedback: nil,
					},
					PayloadType: 109,
				},

				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeH264,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 127,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=127",
						RTCPFeedback: nil,
					},
					PayloadType: 120,
				},

				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeH264,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640032",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 123,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=123",
						RTCPFeedback: nil,
					},
					PayloadType: 118,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     webrtc.MimeTypeVP8,
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "",
						RTCPFeedback: videoRTCPFeedback,
					},
					PayloadType: 96,
				},
				{
					RTPCodecCapability: webrtc.RTPCodecCapability{
						MimeType:     "video/rtx",
						ClockRate:    90000,
						Channels:     0,
						SDPFmtpLine:  "apt=96",
						RTCPFeedback: nil,
					},
					PayloadType: 97,
				},
			},
			HeaderExtensions: nil,
		},
	}
}

// Below code is borrowed from pion/webrtc and a little modified.

type MatchType int

const (
	MatchNone    MatchType = 0
	MatchPartial MatchType = 1
	MatchExact   MatchType = 2
)

// Do a fuzzy find for a codec in the list of codecs. Used to look up a codec
// in an existing list to find a match Returns CodecMatchExact,
// CodecMatchPartial, or CodecMatchNone.
func (r *Registry) FuzzySearch(
	needle transport.Codec,
) (webrtc.RTPCodecParameters, MatchType) {
	haystack := r.getCodecsByMimeType(needle.MimeType)

	needleFmtp := parseFmtp(needle.SDPFmtpLine)

	// First attempt to match on MimeType + SDPFmtpLine
	for _, c := range haystack {
		if strings.EqualFold(c.RTPCodecCapability.MimeType, needle.MimeType) &&
			fmtpConsist(needleFmtp, parseFmtp(c.RTPCodecCapability.SDPFmtpLine)) {
			return c, MatchExact
		}
	}

	// Fallback to just MimeType
	for _, c := range haystack {
		if strings.EqualFold(c.RTPCodecCapability.MimeType, needle.MimeType) {
			return c, MatchPartial
		}
	}

	return webrtc.RTPCodecParameters{}, MatchNone
}

func (r *Registry) RTPHeaderExtensionsForMimeType(mimeType string) []HeaderExtension {
	if TypeFromMimeType(mimeType) == webrtc.RTPCodecTypeAudio {
		return r.Audio.HeaderExtensions
	}

	return r.Video.HeaderExtensions
}

func (r *Registry) getCodecsByMimeType(mimeType string) []webrtc.RTPCodecParameters {
	if TypeFromMimeType(mimeType) == webrtc.RTPCodecTypeAudio {
		return r.Audio.CodecParameters
	}

	return r.Video.CodecParameters
}

func (r *Registry) InterceptorParamsForCodec(codec transport.Codec) (InterceptorParams, error) {
	codecParameters, codecMatch := r.FuzzySearch(codec)
	if codecMatch == MatchNone {
		return InterceptorParams{}, errors.Annotatef(ErrUnsupportedMimeType, "codec: %v", codec)
	}

	var rtcpFeedback []interceptor.RTCPFeedback

	if codecParameters.RTCPFeedback != nil {
		rtcpFeedback := make([]interceptor.RTCPFeedback, len(codecParameters.RTCPFeedback))

		for i, fb := range codecParameters.RTCPFeedback {
			rtcpFeedback[i] = interceptor.RTCPFeedback{
				Type:      fb.Type,
				Parameter: fb.Parameter,
			}
		}
	}

	headerExtensions := r.RTPHeaderExtensionsForMimeType(codec.MimeType)

	var rtpHeaderExtensions []interceptor.RTPHeaderExtension

	if headerExtensions != nil {
		rtpHeaderExtensions = make([]interceptor.RTPHeaderExtension, len(headerExtensions))

		for i, h := range headerExtensions {
			rtpHeaderExtensions[i] = interceptor.RTPHeaderExtension{
				ID:  h.Parameter.ID,
				URI: h.Parameter.URI,
			}
		}
	}

	return InterceptorParams{
		PayloadType:         codecParameters.PayloadType,
		RTCPFeedback:        rtcpFeedback,
		RTPHeaderExtensions: rtpHeaderExtensions,
	}, nil
}

func TypeFromMimeType(mimeType string) webrtc.RTPCodecType {
	if strings.HasPrefix(mimeType, "audio/") {
		return webrtc.RTPCodecTypeAudio
	}

	return webrtc.RTPCodecTypeVideo
}

type InterceptorParams struct {
	PayloadType         webrtc.PayloadType
	RTCPFeedback        []interceptor.RTCPFeedback
	RTPHeaderExtensions []interceptor.RTPHeaderExtension
}
