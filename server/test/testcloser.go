package test

type TestCloser struct {
	cleanups []func() error
}

func (tc *TestCloser) Close() error {
	var err error
	for _, cleanup := range tc.cleanups {
		// cleanup in reverse, like deferred
		err2 := cleanup()
		if err == nil {
			err = err2
		}
	}
	return err
}

func (tc *TestCloser) Add(fn func()) {
	tc.AddFuncErr(func() error {
		fn()
		return nil
	})
}

func (tc *TestCloser) AddFuncErr(cleanup func() error) {
	tc.cleanups = append([]func() error{cleanup}, tc.cleanups...)
}
