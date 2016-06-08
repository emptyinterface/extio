package extio

import "io"

type (
	// A Broadcaster takes a single io.Reader and broadcasts
	// reads from it in parallel to all BroadcasterReaders.
	Broadcaster struct {
		r io.Reader
		// ReadChanLength is the size of the channel that each
		// BroadcasterReader receives reads from.  This allows
		// parallel broadcasting without requiring lock-step
		// synchronization.  This must be set before calling
		// NewReader().  (default: 32)
		ReadChanLength int

		// ReadBufferSize controls the size in bytes of the buffer
		// allocated for each read by the Broadcaster.  It accomplishes
		// buffered reading, as a bufio.ReaderSize does.  This must
		// not be set after calling Broadcast(). (default: 32kb)
		ReadBufferSize int

		brs   []*BroadcasterReader
		abort chan struct{}
	}

	// A BroadcasterReader satisfies the io.ReadCloser interface
	// and receives it's bytes from the Broadcaster's io.Reader
	BroadcasterReader struct {
		b        *Broadcaster
		buf      []byte
		data     chan []byte
		err      chan error
		shutdown chan struct{}
		last     error
	}
)

// NewBroadcaster creates a new Broadcaster from the supplied
// io.Reader and sets ReadChanLength and ReadBufferSize to
// default values.
func NewBroadcaster(r io.Reader) *Broadcaster {

	return &Broadcaster{
		r:              r,
		ReadChanLength: DefaultReadChanLength,
		ReadBufferSize: DefaultBufferSize,
		abort:          make(chan struct{}),
	}

}

// NewReader creates a new BroadcasterReader that can be
// consumed as though it were the original io.Reader
// supplied to the Broadcaster.
func (b *Broadcaster) NewReader() *BroadcasterReader {

	br := &BroadcasterReader{
		b:        b,
		data:     make(chan []byte, b.ReadChanLength),
		err:      make(chan error, 2), // one for EOF, one for ErrClosed
		shutdown: make(chan struct{}),
	}

	b.brs = append(b.brs, br)

	return br

}

// Broadcast initiates reads from the supplied io.Reader
// and sends them to the BroadcasterReaders.  The bytes
// read from the io.Reader are sent over channels so the
// entire sequence is safely concurrent.  It returns any
// error returned by from the underlying io.Reader, except
// io.EOF.  If Abort() was called, returns ErrAborted.
// All errors are passed to all the BroadcasterReaders.
// Broadcast will block until all BroadcasterReaders close.
func (b *Broadcaster) Broadcast() error {

	var err error

	defer func() {
		for _, br := range b.brs {
			close(br.data)
		}
		if err != ErrAborted {
			for _, br := range b.brs {
				br.err <- err
			}
		}
	}()

	for {
		buf := make([]byte, b.ReadBufferSize)
		var n int
		for n < len(buf) && err == nil {
			var nn int
			nn, err = b.r.Read(buf[n:])
			n += nn
		}
		if n > 0 {
			buf = buf[:n]
			for _, br := range b.brs {
				select {
				case br.data <- buf:
				case <-br.shutdown:
					close(br.data)
					close(br.err)
					b.brs = deleteBroadcasterReader(b.brs, br)
				case <-b.abort:
					return ErrAborted
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}

}

// Abort aborts the broadcast.  Causes the Broadcaster and all
// BroadcasterReaders to stop reading and return ErrAborted.
func (b *Broadcaster) Abort() {
	close(b.abort)
}

// Read takes a byte slice and copies broadcast bytes into it
// and returns number of bytes read and any error encountered.
func (br *BroadcasterReader) Read(b []byte) (int, error) {

	if br.last == ErrClosed || br.last == ErrAborted {
		return 0, br.last
	}

LOOP:
	for len(br.buf) < len(b) {
		select {
		case <-br.b.abort:
			br.last = ErrAborted
			return 0, br.last
		case data, open := <-br.data:
			if !open {
				break LOOP
			}
			br.buf = append(br.buf, data...)
		}
	}

	if len(br.buf) > len(b) {
		n := copy(b, br.buf[:len(b)])
		l := copy(br.buf[0:], br.buf[n:])
		br.buf = br.buf[:l]
		return n, nil
	}
	if len(br.buf) > 0 {
		n := copy(b, br.buf)
		br.buf = br.buf[:0]
		return n, nil
	}

	select {
	case br.last = <-br.err:
	default:
	}

	return 0, br.last

}

// Close removes the BroadcasterReader from the broadcast
// stream and causes ErrClosed to be returned on subsequent
// reads. Close will not block until complete.
func (br *BroadcasterReader) Close() error {
	close(br.shutdown)
	br.err <- ErrClosed
	return nil
}

// deletes a BroadcasterReader from a BroadcasterReader slice
// swapping deleted element with first element and slicing off first
// element.  This precise delete strategy allows removing the element
// without disrupting sequential iteration.  *Does not preserve ordering*
func deleteBroadcasterReader(brs []*BroadcasterReader, br *BroadcasterReader) []*BroadcasterReader {
	for i, bbr := range brs {
		if bbr == br {
			if i > 0 {
				brs[i] = brs[0]
			}
			brs = brs[1:]
			break
		}
	}
	return brs
}
