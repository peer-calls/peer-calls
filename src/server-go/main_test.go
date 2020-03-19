package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPanicOnError_panic(t *testing.T) {
	defer func() {
		err, ok := recover().(error)
		require.Equal(t, true, ok)
		require.NotNil(t, err)
		require.Regexp(t, "an error", err.Error())
	}()
	panicOnError(fmt.Errorf("test"), "an error")
}

func TestPanicOnError_noerror(t *testing.T) {
	defer func() {
		err := recover()
		require.Nil(t, err)
	}()
	panicOnError(nil, "an error")
}
