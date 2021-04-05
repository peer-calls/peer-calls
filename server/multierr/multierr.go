package multierr

import (
	e "errors"
	"fmt"
	"strings"

	"github.com/juju/errors"
)

type MultiErr struct {
	firstError error
	errors     []error
}

func New() *MultiErr {
	return &MultiErr{}
}

// Add does nothing when err is nil. It sets the first error if it hasn't been
// set yet.
func (m *MultiErr) Add(err error) {
	if err == nil {
		return
	}

	if m.firstError == nil {
		m.firstError = err
	}

	m.errors = append(m.errors, err)
}

// Err returns all errors that have occurred or nil when no errors had
// occurred. When only one error had occurred, that error is returned. When
// multiple errors have occurred, a new error is returned whose description
// contains the stack traces of all occurring errors.
func (m *MultiErr) Err() error {
	if len(m.errors) <= 1 {
		return m.firstError
	}

	var sb strings.Builder

	err := m.errors[0]

	sb.WriteString("1. ")
	sb.WriteString(errors.ErrorStack(err))

	indexOffset := 2

	for i, err := range m.errors[1:] {
		sb.WriteString(fmt.Sprintf("\n%d. ", i+indexOffset))
		sb.WriteString(errors.ErrorStack(err))
	}

	return errors.Errorf("There were multiple errors:\n%s", sb.String())
}

func Is(err, target error) bool {
	return e.Is(errors.Cause(err), target)
}
