package extio

import (
	"io"
	"sync"
)

type (
	// An AsyncReader takes an io.Reader and buffers it in a goroutine
	// subsequent Read([]byte) calls are populated from buffers sent over
	// an internal channel.
	AsyncReader struct {
		r     io.Reader
		c     chan segment
		abort chan struct{}

		bufs sync.Pool
		buf  []byte

		BufferSize  int
		ChannelSize int
	}
	segment struct {
		b   []byte
		err error
	}
)

// NewAsyncReader creates a new AsyncReader from the supplied io.Reader
// and populates it with defaults
func NewAsyncReader(r io.Reader) *AsyncReader {
	return &AsyncReader{
		r:           r,
		abort:       make(chan struct{}),
		BufferSize:  2 << 20,
		ChannelSize: 32,
	}
}

// Start initializes the goroutine that buffers data from the io.Reader
func (ar *AsyncReader) Start() {
	ar.c = make(chan segment, ar.ChannelSize)
	ar.bufs = sync.Pool{New: func() interface{} { return make([]byte, ar.BufferSize) }}
	go func() {
		defer close(ar.c)
		for {
			buf := ar.bufs.Get().([]byte)
			n, err := io.ReadFull(ar.r, buf)
			select {
			case <-ar.abort:
				return
			case ar.c <- segment{b: buf[:n], err: err}:
			}
			if err != nil {
				// includes io.EOF
				return
			}
		}
	}()
}

// Read takes a byte slice and copies bytes into it
// and returns number of bytes read and any error encountered.
// Will emit io.EOF at completion.
func (ar *AsyncReader) Read(b []byte) (int, error) {
	var (
		s    segment
		open bool
	)
LOOP:
	for len(ar.buf) < len(b) {
		select {
		case <-ar.abort:
			return 0, nil
		case s, open = <-ar.c:
			if !open {
				break LOOP
			}
			if s.err != nil && s.err != io.EOF && s.err != io.ErrUnexpectedEOF {
				return 0, s.err
			}
			ar.buf = append(ar.buf, s.b...)
			ar.bufs.Put(s.b)
		}
	}
	if len(ar.buf) > len(b) {
		n := copy(b, ar.buf[:len(b)])
		l := copy(ar.buf[0:], ar.buf[n:])
		ar.buf = ar.buf[:l]
		return n, nil
	}
	if len(ar.buf) > 0 {
		n := copy(b, ar.buf)
		ar.buf = ar.buf[:0]
		return n, nil
	}
	return 0, io.EOF
}

// Close aborts the buffering goroutine and
// emits no more data on subsequent Read([]byte) calls
func (ar *AsyncReader) Close() error {
	close(ar.abort)
	return nil
}
