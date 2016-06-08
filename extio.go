// Package extio contains extended io strategies
package extio

import "errors"

const (
	// DefaultBufferSize is the default size used for internal buffers (8kb)
	DefaultBufferSize = 8 << 10
	// DefaultReadChanLength is the default size of channels used to buffer communication
	DefaultReadChanLength = 32
	// DefaultWriteChanLength is the default size of channels used to buffer communication
	DefaultWriteChanLength = 32
)

var (
	// ErrAborted indicates an abort was initiated
	ErrAborted = errors.New("aborted")
	// ErrClosed indicates the requested service is closed
	ErrClosed = errors.New("closed")
)
