package codecs

import (
	"strings"

	"github.com/juju/errors"
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
	clockRateVP8    = 90000
	PayloadTypeOpus = 111
	PayloadTypeVP8  = 96
	channelsOpus    = 2
)

func Opus() webrtc.RTPCodecCapability {
	return webrtc.RTPCodecCapability{
		MimeType:     webrtc.MimeTypeOpus,
		ClockRate:    clockRateOpus,
		Channels:     channelsOpus,
		SDPFmtpLine:  "minptime=10;useinbandfec=1",
		RTCPFeedback: nil,
	}
}

func videoRTCPFeedback() []webrtc.RTCPFeedback {
	return []webrtc.RTCPFeedback{
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
}

func VP8() webrtc.RTPCodecCapability {
	return webrtc.RTPCodecCapability{
		MimeType:     webrtc.MimeTypeVP8,
		ClockRate:    clockRateVP8,
		Channels:     0,
		SDPFmtpLine:  "",
		RTCPFeedback: videoRTCPFeedback(),
	}
}

func NewRegistryDefault() *Registry {
	return &Registry{
		Audio: Props{
			CodecParameters: []webrtc.RTPCodecParameters{
				{
					RTPCodecCapability: Opus(),
					PayloadType:        PayloadTypeOpus,
				},
			},
			HeaderExtensions: nil,
		},
		Video: Props{
			CodecParameters: []webrtc.RTPCodecParameters{
				{
					RTPCodecCapability: VP8(),
					PayloadType:        PayloadTypeVP8,
				},
			},
			HeaderExtensions: nil,
		},
	}
}

// Below code is borrowed from pion/webrtc and a little modified.

type CodecMatchType int

const (
	CodecMatchNone    CodecMatchType = 0
	CodecMatchPartial CodecMatchType = 1
	CodecMatchExact   CodecMatchType = 2
)

// Do a fuzzy find for a codec in the list of codecs. Used to look up a codec
// in an existing list to find a match Returns CodecMatchExact,
// CodecMatchPartial, or CodecMatchNone.
func (r *Registry) FuzzySearch(
	needle webrtc.RTPCodecParameters,
) (webrtc.RTPCodecParameters, CodecMatchType) {
	haystack := r.getCodecsByMimeType(needle.MimeType)

	needleFmtp := parseFmtp(needle.RTPCodecCapability.SDPFmtpLine)

	// First attempt to match on MimeType + SDPFmtpLine
	for _, c := range haystack {
		if strings.EqualFold(c.RTPCodecCapability.MimeType, needle.RTPCodecCapability.MimeType) &&
			fmtpConsist(needleFmtp, parseFmtp(c.RTPCodecCapability.SDPFmtpLine)) {
			return c, CodecMatchExact
		}
	}

	// Fallback to just MimeType
	for _, c := range haystack {
		if strings.EqualFold(c.RTPCodecCapability.MimeType, needle.RTPCodecCapability.MimeType) {
			return c, CodecMatchPartial
		}
	}

	return webrtc.RTPCodecParameters{}, CodecMatchNone
}

func (r *Registry) FindByMimeType(mimeType string) (webrtc.RTPCodecParameters, bool) {
	haystack := r.getCodecsByMimeType(mimeType)

	// Fallback to just MimeType
	for _, c := range haystack {
		if strings.EqualFold(c.RTPCodecCapability.MimeType, mimeType) {
			return c, true
		}
	}

	return webrtc.RTPCodecParameters{}, false
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

func (r *Registry) InterceptorParamsForMimeType(mimeType string) (InterceptorParams, error) {
	codecParameters, ok := r.FindByMimeType(mimeType)
	if !ok {
		return InterceptorParams{}, errors.Annotate(ErrUnsupportedMimeType, mimeType)
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

	headerExtensions := r.RTPHeaderExtensionsForMimeType(mimeType)

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
