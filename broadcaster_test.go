package extio

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

type (
	sleepyReader struct {
		*bytes.Reader
	}
	errorReader struct {
		err error
	}
)

func (r *sleepyReader) Read(b []byte) (int, error) {
	time.Sleep(100 * time.Millisecond)
	return r.Reader.Read(b)
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestBroadcaster(t *testing.T) {

	testdata := make([]byte, (2<<20)+21)
	rand.Read(testdata)

	b := NewBroadcaster(bytes.NewReader(testdata))

	var (
		outputs = []*bytes.Buffer{
			&bytes.Buffer{},
			&bytes.Buffer{},
			&bytes.Buffer{},
		}
		wg sync.WaitGroup
	)

	for _, out := range outputs {
		wg.Add(1)
		out := out
		br := b.NewReader()
		go func() {
			defer wg.Done()
			if _, err := io.Copy(out, br); err != nil {
				t.Error(err)
			}
		}()
	}

	if err := b.Broadcast(); err != nil {
		t.Error(err)
	}

	wg.Wait()

	for i, out := range outputs {
		if n := bytes.Compare(out.Bytes(), testdata); n != 0 {
			t.Errorf("%d reader reported %d bytes diff", i, n)
		}
	}

}

func TestBroadcasterAbort(t *testing.T) {

	b := NewBroadcaster(&sleepyReader{bytes.NewReader(data)})

	var (
		outputs = []*bytes.Buffer{
			&bytes.Buffer{},
			&bytes.Buffer{},
			&bytes.Buffer{},
		}
		wg sync.WaitGroup
	)

	for _, out := range outputs {
		wg.Add(1)
		out := out
		br := b.NewReader()
		go func() {
			defer wg.Done()
			if _, err := io.Copy(out, br); err != ErrAborted {
				t.Errorf("Expected %q, got %q", ErrAborted, err)
			}
			if _, err := br.Read(nil); err != ErrAborted {
				t.Errorf("Expected %q, got %q", ErrAborted, err)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := b.Broadcast(); err != ErrAborted {
			t.Error(err)
		}
	}()

	// abort can be called with/without starting
	b.Abort()

	wg.Wait()

	for _, out := range outputs {
		if out.Len() != 0 {
			t.Errorf("expected zero bytes, got %d", out.Len())
		}
	}

}

func TestBroadcasterClose(t *testing.T) {

	var data [32]byte

	b := NewBroadcaster(bytes.NewReader(data[:]))
	br := b.NewReader()

	go b.Broadcast()

	var buf [32]byte
	// read out all data
	if _, err := br.Read(buf[:]); err != nil {
		t.Error(err)
	}

	if _, err := br.Read(buf[:]); err != io.EOF {
		t.Errorf("Expected %q, got %q", io.EOF, err)
	}

	if err := br.Close(); err != nil {
		t.Error(err)
	}

	if _, err := br.Read(buf[:]); err != ErrClosed {
		t.Errorf("Expected %q, got %q", ErrClosed, err)
	}

}

func TestBroadcasterCloseDuringRead(t *testing.T) {

	b := NewBroadcaster(&sleepyReader{bytes.NewReader(data)})
	b.ReadBufferSize = 8
	b.ReadChanLength = 1

	br := b.NewReader()

	go func() {
		if err := b.Broadcast(); err != nil {
			t.Error(err)
		}
	}()

	var buf [2]byte
	if _, err := br.Read(buf[:]); err != nil {
		t.Error(err)
	}
	// close to remove from further reads
	if err := br.Close(); err != nil {
		t.Error(err)
	}

	time.Sleep(300 * time.Millisecond) // wait for our sleepy reader

	if len(b.brs) != 0 {
		t.Errorf("Expected %d readers, got %d", 0, len(b.brs))
	}

}

func TestBroadcasterErrors(t *testing.T) {

	testError := errors.New("test")

	b := NewBroadcaster(&errorReader{err: testError})

	br := b.NewReader()

	go func() {
		if err := b.Broadcast(); err != testError {
			t.Errorf("Expected %q, got %q", testError, err)
		}
	}()

	var buf [2]byte
	if _, err := br.Read(buf[:]); err != testError {
		t.Errorf("Expected %q, got %q", testError, err)
	}

	// close should not emit read error
	if err := br.Close(); err != nil {
		t.Error(err)
	}

}

func TestDeleteBroadcasterReader(t *testing.T) {

	b := NewBroadcaster(bytes.NewReader([]byte{}))

	orig := []*BroadcasterReader{
		b.NewReader(),
		b.NewReader(),
		b.NewReader(),
	}

	testset := append([]*BroadcasterReader(nil), orig...)

	testset = deleteBroadcasterReader(testset, orig[1])
	for _, br := range testset {
		if br == orig[1] {
			t.Error("Failed to delete middle element")
		}
	}

	testset = deleteBroadcasterReader(testset, orig[1])
	for _, br := range testset {
		if br == orig[1] {
			t.Error("Failed to delete last element")
		}
	}

	testset = deleteBroadcasterReader(testset, orig[0])
	for _, br := range testset {
		if br == orig[0] {
			t.Error("Failed to delete first/only element")
		}
	}

}

func BenchmarkBroadcaster(b *testing.B) {

	const (
		readerCt = 1
		dataSize = 32 << 20
	)

	testdata := make([]byte, dataSize)
	rand.Read(testdata)
	b.SetBytes(dataSize)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		bc := NewBroadcaster(bytes.NewReader(testdata))

		var wg sync.WaitGroup
		wg.Add(readerCt)

		for i := 0; i < readerCt; i++ {
			br := bc.NewReader()
			go func() {
				defer wg.Done()
				io.Copy(ioutil.Discard, br)
			}()
		}

		b.StartTimer()
		bc.Broadcast()
		wg.Wait()
	}

}
