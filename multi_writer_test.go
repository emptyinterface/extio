package extio

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"testing"
)

type (
	testOKWriteCloser struct {
		bytes.Buffer
	}
	testErrorWriteCloser struct {
		bytes.Buffer
	}
	testErrorWriter struct {
		bytes.Buffer
	}
	testShortWriter struct {
		bytes.Buffer
	}
)

var (
	closeErr = errors.New("close err")
	writeErr = errors.New("write err")
)

func (_ *testOKWriteCloser) Close() error              { return nil }
func (_ *testErrorWriteCloser) Close() error           { return closeErr }
func (_ *testErrorWriter) Write(_ []byte) (int, error) { return 0, writeErr }
func (_ *testShortWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }

func TestMultiWriterOne(t *testing.T) {

	buf := &testOKWriteCloser{}
	mw := NewMultiWriter(buf)

	n, err := mw.Write(data)
	if err != nil {
		t.Error(err)
	}
	if n != len(data) {
		t.Errorf("Short write!  expected %d, got %d", data, n)
	}

	if err := mw.Close(); err != nil {
		t.Error(err)
	}

	n, err = mw.Write(data)
	if n != 0 {
		t.Errorf("Expected 0 bytes on Write after close, got %d\n", n)
	}
	if err != ErrClosed {
		t.Errorf("Expected %q, got %q", ErrClosed, err)
	}

	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("data did not match buffer")
	}

}

func TestMultiWriterErrors(t *testing.T) {

	// test error on close
	mw := NewMultiWriter(&testErrorWriteCloser{})
	mw.Write(data)
	if err := mw.Close(); err != closeErr {
		t.Errorf("Expected %q, got %q", closeErr, err)
	}

	// test error on write
	mw = NewMultiWriter(&testErrorWriter{})
	mw.WriteChanLength = 0 // cause blocking so error surfaces after one write
	mw.Write(data)         // first write
	if n, err := mw.Write(data); err != writeErr {
		t.Errorf("Expected %q, got %q", writeErr, err)
	} else if n != 0 {
		t.Errorf("Expected 0 bytes on Write, got %d\n", n)
	}

	// test short write
	mw = NewMultiWriter(&testShortWriter{})
	mw.WriteChanLength = 0 // cause blocking so error surfaces after one write
	mw.Write(data)         // first write
	if n, err := mw.Write(data); err != io.ErrShortWrite {
		t.Errorf("Expected %q, got %q", io.ErrShortWrite, err)
	} else if n != 0 {
		t.Errorf("Expected 0 bytes on Write, got %d\n", n)
	}

}

func TestMultiWriterRange(t *testing.T) {

	for i := 0; i < 30; i++ {

		var bufs []io.Writer
		for j := 0; j < i; j++ {
			bufs = append(bufs, &bytes.Buffer{})
		}

		mw := NewMultiWriter(bufs...)

		for j := 0; j < i; j++ {
			n, err := mw.Write(data)
			if err != nil {
				t.Error(err)
			}
			if n != len(data) {
				t.Errorf("Short write!  expected %d, got %d", data, n)
			}
		}

		mw.Close()

		for j := 0; j < i; j++ {
			for _, buf := range bufs {
				output := buf.(*bytes.Buffer).Bytes()
				if expected := bytes.Repeat(data, i); !bytes.Equal(expected, output) {
					t.Errorf("data mismatch")
				}
			}
		}

	}

}

func BenchmarkMultiWriter(b *testing.B) {

	mw := NewMultiWriter(ioutil.Discard)

	b.SetBytes(int64(len(data)))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}

	mw.Close()

}

func BenchmarkStdlibMultiWriter(b *testing.B) {

	mw := io.MultiWriter(ioutil.Discard)

	b.SetBytes(int64(len(data)))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}

}
