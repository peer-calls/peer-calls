package test

import "github.com/juju/errors"

// TODO use t.Cleanup instead.
type Closer struct {
	cleanups []func() error
}

func (tc *Closer) Close() error {
	var err error

	for _, cleanup := range tc.cleanups {
		// cleanup in reverse, like deferred
		err2 := cleanup()

		if err == nil {
			err = err2
		}
	}

	return errors.Trace(err)
}

func (tc *Closer) Add(fn func()) {
	tc.AddFuncErr(func() error {
		fn()

		return nil
	})
}

func (tc *Closer) AddFuncErr(cleanup func() error) {
	tc.cleanups = append([]func() error{cleanup}, tc.cleanups...)
}
