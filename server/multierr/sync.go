package multierr

import "sync"

// Sync wraps MultiErr and is safe for concurrent use.
type Sync struct {
	mu       sync.Mutex
	multierr MultiErr
}

// Add wraps MultiErr.Add with a mutex lock.
func (s *Sync) Add(err error) {
	s.mu.Lock()

	s.multierr.Add(err)

	s.mu.Unlock()
}

// Add wraps MultiErr.Err with a mutex lock.
func (s *Sync) Err() error {
	s.mu.Lock()

	err := s.multierr.Err()

	s.mu.Unlock()

	return err
}
