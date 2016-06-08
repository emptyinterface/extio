package main

import (
	"crypto"
	"fmt"
	"hash"
	"io"
	"time"

	_ "crypto/md5"
	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"

	_ "golang.org/x/crypto/md4"
	_ "golang.org/x/crypto/ripemd160"
	_ "golang.org/x/crypto/sha3"

	"github.com/emptyinterface/extio"
)

func main() {

	var (
		data [32 << 20]byte

		hashes = []func() hash.Hash{
			crypto.MD4.New,
			crypto.MD5.New,
			crypto.SHA1.New,
			crypto.SHA224.New,
			crypto.SHA256.New,
			crypto.SHA384.New,
			crypto.SHA512.New,
			crypto.RIPEMD160.New,
			crypto.SHA3_224.New,
			crypto.SHA3_256.New,
			crypto.SHA3_384.New,
			crypto.SHA3_512.New,
			crypto.SHA512_224.New,
			crypto.SHA512_256.New,
		}
	)

	{
		start := time.Now()
		var hashesw []io.Writer
		for _, h := range hashes {
			hashesw = append(hashesw, h())
		}
		mw := io.MultiWriter(hashesw...)
		mw.Write(data[:])
		for _, w := range hashesw {
			fmt.Printf("%.8x\n", w.(hash.Hash).Sum(nil))
		}
		fmt.Println("32mb processed by", len(hashes), "hashes using io.MultiWriter", time.Since(start))
	}

	{
		start := time.Now()
		var hashesw []io.Writer
		for _, h := range hashes {
			hashesw = append(hashesw, h())
		}
		emw := extio.NewMultiWriter(hashesw...)
		emw.Write(data[:])
		emw.Close()
		for _, w := range hashesw {
			fmt.Printf("%.8x\n", w.(hash.Hash).Sum(nil))
		}
		fmt.Println("32mb processed by", len(hashes), "hashes using extio.MultiWriter", time.Since(start))
	}

}
