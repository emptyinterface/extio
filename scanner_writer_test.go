package extio

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"testing"
)

// tests ScannerWriter parity with bufio.Scanner
func TestScannerWriter(t *testing.T) {

	for _, splitFunc := range []bufio.SplitFunc{
		bufio.ScanLines,
		bufio.ScanWords,
		bufio.ScanRunes,
		bufio.ScanBytes,
	} {

		sc := bufio.NewScanner(bytes.NewReader(data))
		sc.Split(splitFunc)

		var prev []byte

		w := NewScannerWriter(splitFunc, 1<<10, func(token []byte) error {
			if sc.Scan() {
				if !bytes.Equal(sc.Bytes(), token) {
					return fmt.Errorf("After %q Expected: %q, got %q", prev, sc.Bytes(), token)
				}
				prev = sc.Bytes()
			}
			if sc.Err() != nil {
				return sc.Err()
			}
			return nil
		})

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			data := []byte(data)
			for len(data) > 100 {
				n := rand.Intn(len(data))
				if _, err := w.Write(data[:n]); err != nil {
					t.Error(err)
				}
				data = data[n:]
			}
			if n, err := w.Write(data); err != nil {
				t.Error(err)
			} else if n != len(data) {
				t.Errorf("Expected %d bytes written, got %d", len(data), n)
			}
			if err := w.Flush(); err != nil {
				t.Error(err)
			}
			if err := w.Close(); err != nil {
				t.Error(err)
			}
			n, err := w.Write(data)
			if n != 0 {
				t.Errorf("Expected 0 bytes on Write after close, got %d\n", n)
			}
			if err != ErrClosed {
				t.Errorf("Expected %q, got %q", ErrClosed, err)
			}
		}()

		wg.Wait()

	}

}

func TestScannerWriterFlush(t *testing.T) {

	var (
		expected = []string{"a", "bc"}
		i        int
	)

	w := NewScannerWriter(bufio.ScanWords, 1<<10, func(token []byte) error {
		if string(token) != expected[i] {
			t.Errorf("Expected %q, got %q", expected[i], string(token))
		}
		i++
		return nil
	})
	if _, err := w.Write([]byte("a b")); err != nil {
		t.Error(err)
	}
	if _, err := w.Write([]byte("c")); err != nil {
		t.Error(err)
	}
	if err := w.Flush(); err != nil {
		t.Error(err)
	}
	if err := w.Close(); err != nil {
		t.Error(err)
	}

	if err := w.Flush(); err != ErrClosed {
		t.Errorf("Expected %q, got %q", ErrClosed, err)
	}

	if err := w.Close(); err != ErrClosed {
		t.Errorf("Expected %q, got %q", ErrClosed, err)
	}

}

func TestScannerWriterErrors(t *testing.T) {

	var (
		splitErr     = errors.New("split err")
		tokenErr     = errors.New("token err")
		errSplitFunc = func(_ []byte, _ bool) (int, []byte, error) { return 0, nil, splitErr }
		errTokenFunc = func(token []byte) error { return tokenErr }
	)

	// test token func error
	w := NewScannerWriter(bufio.ScanWords, 1<<10, errTokenFunc)
	if n, err := w.Write([]byte("a b c")); err != tokenErr {
		t.Errorf("Expected %q, got %q", tokenErr, err)
	} else if n != 0 {
		t.Errorf("Expected %d bytes read, got %d", 0, n)
	}
	if err := w.Flush(); err != nil {
		t.Error(err)
	}
	if err := w.Close(); err != nil {
		t.Error(err)
	}

	// test split func error
	w = NewScannerWriter(errSplitFunc, 1<<10, func(_ []byte) error { return nil })
	if n, err := w.Write([]byte("a b c")); err != splitErr {
		t.Errorf("Expected %q, got %q", splitErr, err)
	} else if n != 0 {
		t.Errorf("Expected %d bytes read, got %d", 0, n)
	}
	if err := w.Flush(); err != nil {
		t.Error(err)
	}
	if err := w.Close(); err != nil {
		t.Error(err)
	}

	// test buffer exceeded
	w = NewScannerWriter(bufio.ScanWords, 1, func(_ []byte) error { return nil })
	if n, err := w.Write([]byte("ab")); err != io.ErrShortBuffer {
		t.Errorf("Expected %q, got %q", io.ErrShortBuffer, err)
	} else if n != 0 {
		t.Errorf("Expected %d bytes read, got %d", 0, n)
	}
	if err := w.Flush(); err != nil {
		t.Error(err)
	}
	if err := w.Close(); err != nil {
		t.Error(err)
	}

	// test flush token, split funcs errors
	w = NewScannerWriter(bufio.ScanWords, 1<<10, func(_ []byte) error { return nil })
	if _, err := w.Write([]byte("ab")); err != nil {
		t.Error(nil)
	}
	w.tokenFunc = errTokenFunc
	if err := w.Flush(); err != tokenErr {
		t.Errorf("Expected %q, got %q", tokenErr, err)
	}
	if _, err := w.Write([]byte("ab")); err != nil {
		t.Error(nil)
	}
	w.splitFunc = errSplitFunc
	if err := w.Flush(); err != splitErr {
		t.Errorf("Expected %q, got %q", splitErr, err)
	}
	if err := w.Close(); err != splitErr {
		t.Errorf("Expected %q, got %q", splitErr, err)
	}

}

func BenchmarkScannerWriterScan7Bytes(b *testing.B) {
	runBenchmarkScannerWriter([]byte("Gibbons"), b)
}
func BenchmarkScannerWriterScan22Bytes(b *testing.B) {
	runBenchmarkScannerWriter([]byte("Gibbons (/ˈɡɪbənz/[3])"), b)
}
func BenchmarkScannerWriterScan31Bytes(b *testing.B) {
	runBenchmarkScannerWriter([]byte("Gibbons (/ˈɡɪbənz/[3]) are apes"), b)
}
func BenchmarkScannerWriterScan57Bytes(b *testing.B) {
	runBenchmarkScannerWriter([]byte("Gibbons (/ˈɡɪbənz/[3]) are apes in the family Hylobatidae"), b)
}
func BenchmarkScannerWriterScan75Bytes(b *testing.B) {
	runBenchmarkScannerWriter([]byte("Gibbons (/ˈɡɪbənz/[3]) are apes in the family Hylobatidae /ˌhaɪloʊbəˈtaɪdeɪ"), b)
}
func BenchmarkScannerWriterScan1572Bytes(b *testing.B) {
	runBenchmarkScannerWriter(data, b)
}

func runBenchmarkScannerWriter(body []byte, b *testing.B) {

	w := NewScannerWriter(bufio.ScanWords, 1<<10, func(_ []byte) error { return nil })

	b.SetBytes(int64(len(body)))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Write(body)
	}

	b.StopTimer()

	w.Close()

}

var (
	data = []byte(`

	https://en.wikipedia.org/wiki/Gibbon

	Gibbons (/ˈɡɪbənz/[3]) are apes in the family Hylobatidae /ˌhaɪloʊbəˈtaɪdeɪ, -diː/[4].
	The family historically contained one genus, but now is split into four genera and 17
	species. Gibbons occur in tropical and subtropical rainforests from eastern Bangladesh
	and northeast India to southern China and Indonesia (including the islands of Sumatra,
	Borneo, and Java).

	Also called the smaller apes,[5] gibbons differ from great apes (chimpanzees, bonobos,
	gorillas, orangutans, and humans) in being smaller, exhibiting low sexual dimorphism
	and not making nests. In certain anatomical details they superficially more closely
	resemble monkeys than great apes do, but like all apes, gibbons are tailless. Gibbons
	also display pair-bonding, maintaining the same mate for life, unlike most of the great
	apes (this has been disputed by Palombit and others, who have found that gibbons might
	be socially monogamous, with occasional "divorce", but not sexually monogamous[6][7]).
	Gibbons are masters of their primary mode of locomotion, brachiation, swinging from
	branch to branch for distances of up to 15 m (50 ft), at speeds as high as 55 km/h
	(34 mph). They can also make leaps of up to 8 m (26 ft), and walk bipedally with their
	arms raised for balance. They are the fastest and most agile of all tree-dwelling,
	nonflying mammals.[8]

	Depending on species and sex, gibbons' fur coloration varies from dark to light
	brown shades, and any shade between black and white. Seeing a completely "white"
	gibbon is rare.

`)
)
