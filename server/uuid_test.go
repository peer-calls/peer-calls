package server

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUUIDBase62(t *testing.T) {
	value := NewUUIDBase62()
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
		NewUUIDBase62()
	}
}

func TestTime(t *testing.T) {
	NewUUIDBase62()
}
