package extio

import (
	"bytes"
	"crypto/rand"
	"io"
	"io/ioutil"
	mr "math/rand"
	"testing"
)

func TestAsyncReader(t *testing.T) {

	for i := 0; i < 200; i++ {
		buf := make([]byte, 2<<10+mr.Intn(32<<10))
		rand.Read(buf)

		ar := NewAsyncReader(bytes.NewReader(buf))
		ar.BufferSize = mr.Intn(64 << 10)
		ar.ChannelSize = mr.Intn(128)
		ar.Start()

		data, err := ioutil.ReadAll(ar)
		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal(buf, data) {
			t.Error("buf/data mismatch")
		}

	}

}

func BenchmarkReader(b *testing.B) {
	buf := make([]byte, 8<<20)
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(buf)
		io.Copy(ioutil.Discard, r)
	}
}

func BenchmarkAsyncReader(b *testing.B) {
	buf := make([]byte, 8<<20)
	b.SetBytes(int64(len(buf)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ar := NewAsyncReader(bytes.NewReader(buf))
		ar.Start()
		io.Copy(ioutil.Discard, ar)
	}
}
