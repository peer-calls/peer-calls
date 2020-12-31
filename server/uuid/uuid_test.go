package uuid

import (
	"testing"

	"github.com/google/uuid"
)

func TestNew(t *testing.T) {
	value := New()
	t.Log(value)
}

func BenchmarkNewUUID_normal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uuid.New().String()
	}
}

func BenchmarkNewUUID_base62(b *testing.B) {
	for i := 0; i < b.N; i++ {
		New()
	}
}

func TestTime(t *testing.T) {
	t.Parallel()

	New()
}
