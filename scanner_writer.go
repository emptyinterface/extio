package extio

import (
	"bufio"
	"io"
)

type (
	// ScannerWriter satisfies the io.WriteCloser interface and
	// turns a series of writes into a stream of tokens that can
	// be processed by a callback.
	ScannerWriter struct {
		buf        []byte
		maxBufSize int

		closed bool

		splitFunc bufio.SplitFunc
		tokenFunc func(token []byte) error
	}
)

// NewScannerWriter creates a new ScannerWriter.  Arguments are
// a function that satifies the bufio.SplitFunc type.  This is
// used to parse the incoming byte stream.  A maxBufSize, which
// determines how far to read into the byte stream without finding
// a token, before throwing an io.ErrShortBuffer.  And a tokenFunc
// that takes the next token identified by splitFunc, and returns
// an error. An error returned by a splitFunc is returned to the
// caller of Write().
func NewScannerWriter(splitFunc bufio.SplitFunc, maxBufSize int, tokenFunc func([]byte) error) *ScannerWriter {
	return &ScannerWriter{
		splitFunc:  splitFunc,
		tokenFunc:  tokenFunc,
		maxBufSize: maxBufSize,
	}
}

// Write writes the contents of data to the buffer and immediately
// parses the buffer for as many tokens as splitFunc identifies.
// Any remaining data is left in the buffer until the next Write
// or Flush.  Returns number of bytes written and any error.
func (sc *ScannerWriter) Write(data []byte) (int, error) {

	if sc.closed {
		return 0, ErrClosed
	}

	dataLen := len(data)

	if sc.buf != nil {
		tmp := make([]byte, len(sc.buf)+len(data))
		copy(tmp, sc.buf)
		copy(tmp[len(sc.buf):], data)
		data = tmp
		sc.buf = nil
	}

	for len(data) > 0 {

		adv, token, err := sc.splitFunc(data, false)
		if err != nil {
			return 0, err
		}

		if token == nil {
			if adv == 0 {
				if len(data) > sc.maxBufSize {
					return 0, io.ErrShortBuffer
				}
				sc.buf = make([]byte, len(data))
				copy(sc.buf, data)
				return dataLen, nil
			}
		} else if err := sc.tokenFunc(token); err != nil {
			return 0, err
		}

		if adv > 0 {
			data = data[adv:]
		}

	}

	return dataLen, nil

}

// Flush fluses the contents of the buffer to the splitFunc
// signalling EOF.
func (sc *ScannerWriter) Flush() error {

	if sc.closed {
		return ErrClosed
	}

	if len(sc.buf) == 0 {
		return nil
	}

	_, token, err := sc.splitFunc(sc.buf, true)
	if err != nil {
		return err
	}

	sc.buf = nil

	if len(token) > 0 {
		if err := sc.tokenFunc(token); err != nil {
			return err
		}
	}

	return nil

}

// Close closes the ScannerWriter after calling Flush().
// Any subsequent writes will return ErrClosed.
func (sc *ScannerWriter) Close() error {

	if sc.closed {
		return ErrClosed
	}

	if err := sc.Flush(); err != nil {
		return err
	}

	sc.closed = true

	return nil

}
