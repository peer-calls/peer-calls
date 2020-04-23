package basen_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/peer-calls/peer-calls/server/basen"
)

func TestNewUUIDBase62(t *testing.T) {
	value := basen.NewUUIDBase62()
	t.Log(value)
}

var s string

func BenchmarkNewUUID_normal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s = uuid.New().String()
	}
}

func BenchmarkNewUUID_base62(b *testing.B) {
	for i := 0; i < b.N; i++ {
		basen.NewUUIDBase62()
	}
}

func TestTime(t *testing.T) {
	basen.NewUUIDBase62()
}
