package extio

import (
	"io"
	"sync"
)

type (
	// A MultiWriter satisfies the io.WriteCloser interface and
	// allows for multiple io.Writers to be written to concurrently
	// from a single write.  The functionality is similar to the
	// io.MultiWriter except that each io.Writer receives it's data
	// in a separate goroutine.
	MultiWriter struct {
		writers []*mwWriter

		WriteChanLength int

		inited bool
		closed bool
		err    chan error
		wg     sync.WaitGroup
	}

	mwWriter struct {
		w  io.Writer
		wc chan []byte
	}
)

// NewMultiWriter creates a MultiWriter from the io.Writer(s)
// specified as args.  This only creates the data structure
// and does not initialize any goroutines.
func NewMultiWriter(ws ...io.Writer) *MultiWriter {

	mw := &MultiWriter{
		WriteChanLength: DefaultWriteChanLength,
		err:             make(chan error, 1),
	}

	for _, w := range ws {
		mw.writers = append(mw.writers, &mwWriter{w: w})
	}

	return mw

}

// Handles the initialization of channels and goroutines
// required for the concurrent distribution of writes.
func (mw *MultiWriter) init() {

	mw.inited = true

	for _, mww := range mw.writers {

		mww.wc = make(chan []byte, mw.WriteChanLength)
		mw.wg.Add(1)

		go func(mww *mwWriter) {
			defer func() {
				if wc, ok := mww.w.(io.WriteCloser); ok {
					if err := wc.Close(); err != nil {
						mw.err <- err
					}
				}
				mw.wg.Done()
			}()
			for data := range mww.wc {
				if n, err := mww.w.Write(data); err != nil {
					mw.err <- err
					return
				} else if n < len(data) {
					mw.err <- io.ErrShortWrite
					return
				}
			}
		}(mww)

	}

}

// Write takes a byte slice and writes it to each io.Writer
// of the MultiWriter.  This happens through channels to allow
// each io.Writer to process the data concurrently.  Any
// alteration of the byte slice by any io.Writers will produce
// undefined behavior.  Write returns the number of bytes written
// and any error returned by an io.Writer since the first Write.
// Due to the buffering of channels, this error is not guaranteed
// to be present for the write that it fails on.
func (mw *MultiWriter) Write(data []byte) (int, error) {

	if mw.closed {
		return 0, ErrClosed
	}

	if !mw.inited {
		mw.init()
	}

	for _, mww := range mw.writers {
		select {
		case mww.wc <- data:
		case err := <-mw.err:
			return 0, err
		}
	}

	return len(data), nil

}

// Close closes each data channel.  After the remaining
// data is drained from the data channels, each io.Writer is
// checked for a `Close() error` method.  If the method is
// found it is called.  This method blocks until all io.Writers
// have completed consuming their data channels, and optionally
// closed.  The first error encountered is returned, or nil if none.
func (mw *MultiWriter) Close() error {

	mw.closed = true

	if mw.inited {
		for _, mww := range mw.writers {
			close(mww.wc)
		}

		mw.wg.Wait()
		close(mw.err)

		return <-mw.err
	}

	return nil

}
