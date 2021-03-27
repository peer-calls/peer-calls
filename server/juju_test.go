package server_test

import (
	native "errors"
	"fmt"
	"testing"

	"github.com/juju/errors"
)

var errTest = native.New("test")

func errorNoJuju(n int) error {
	err := fmt.Errorf("err: %w", errTest)

	for i := 0; i < n; i++ {
		err = fmt.Errorf("err %d: %w", i, err)
	}

	return err
}

func recursiveErrorJuju(n int) error {
	err := errors.Annotate(errTest, "err")

	for i := 0; i < n; i++ {
		err = errors.Annotatef(err, "err: %d", i)
	}

	return errors.Trace(err)
}

func BenchmarkNoJuju(b *testing.B) {
	err := errorNoJuju(b.N)
	fmt.Printf("%s\n", native.Unwrap(err))
}

func BenchmarkJuju(b *testing.B) {
	err := recursiveErrorJuju(b.N)
	fmt.Printf("%s\n", errors.Cause(err))
}

func addDefer(a, b int) (r int) {
	defer func() {
		r = a + b
	}()

	return
}

func addNoDefer(a, b int) (r int) {
	r = a + b

	return
}

func BenchmarkDefer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addDefer(i, i+1)
	}
}

func BenchmarkNoDefer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addNoDefer(i, i+1)
	}
}
