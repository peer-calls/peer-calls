package promise_test

import (
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/promise"
	"github.com/stretchr/testify/assert"
)

func TestPromise_Resolve(t *testing.T) {
	t.Parallel()

	p := promise.New()

	go func() {
		p.Resolve()
	}()

	err := p.Wait()
	assert.Nil(t, err)
}

func TestPromise_Reject(t *testing.T) {
	t.Parallel()

	p := promise.New()

	errTest := errors.New("test")

	go func() {
		p.Reject(errTest)
	}()

	err := p.Wait()
	assert.Equal(t, errTest, errors.Cause(err))
}
