package stats

import (
	"time"
)

// nolint:gochecknoglobals
var offsetNTP = uint64(
	time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Sub(
		time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
	) / time.Second,
)

const nanosecond uint64 = 1e9

type NTPTime uint64

func NewNTPTime(t time.Time) NTPTime {
	var seconds uint64
	var fraction uint64
	unixNano := uint64(t.UnixNano())

	seconds = unixNano / nanosecond
	// seconds += 0x83AA7E80 // offset in seconds between unix epoch and ntp epoch
	seconds += offsetNTP
	seconds <<= 32

	fraction = unixNano % nanosecond
	fraction <<= 32
	fraction /= 1e9

	return NTPTime(seconds + fraction)
}

func (t NTPTime) Time() time.Time {
	// nolint:gomnd
	seconds := uint64(t) >> 32
	seconds -= offsetNTP

	// nolint:gomnd
	fraction := uint64(t) & 0xFFFF_FFFF
	fraction *= nanosecond
	fraction >>= 32

	nanos := seconds*nanosecond + fraction

	return time.Unix(0, int64(nanos)).UTC()
}

func (t NTPTime) Middle() uint32 {
	// nolinth: gomnd
	return uint32(t << 16)
}
