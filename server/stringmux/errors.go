package stringmux

import "github.com/juju/errors"

var (
	ErrConnAlreadyExists = errors.New("connection already exists")
	ErrConnNotFound      = errors.New("connection not found")
)
