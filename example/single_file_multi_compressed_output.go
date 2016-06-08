package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/emptyinterface/extio"
)

func main() {

	const fileSize = 8 << 20

	var data [fileSize]byte
	rand.Read(data[:])

	file, err := ioutil.TempFile("", "extio_test_file")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := file.Write(data[:]); err != nil {
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
	defer os.Remove(file.Name())

	func() {
		// Broadcaster example
		start := time.Now()

		file, err := os.OpenFile(file.Name(), os.O_RDONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		b := extio.NewBroadcaster(file)

		cmds := []*exec.Cmd{
			exec.Command("sh", "-c", "gzip -9 -c > /dev/null"),
			exec.Command("sh", "-c", "bzip2 -9 -z -c > /dev/null"),
			exec.Command("sh", "-c", "xz -9 -z -c > /dev/null"),
		}

		for _, cmd := range cmds {
			cmd.Stdin = b.NewReader()
		}

		for _, cmd := range cmds {
			if err := cmd.Start(); err != nil {
				log.Fatal(err)
			}
		}

		if err := b.Broadcast(); err != nil {
			log.Fatal(err)
		}

		for _, cmd := range cmds {
			if err := cmd.Wait(); err != nil {
				log.Fatal(err)
			}
		}

		fmt.Println("extio.Broadcaster compressed 8mb in gzip, bzip2, and xz in", time.Since(start))
	}()

	func() {
		// io.Pipe/io.Copy example
		start := time.Now()

		cmds := []*exec.Cmd{
			exec.Command("sh", "-c", "gzip -9 -c > /dev/null"),
			exec.Command("sh", "-c", "bzip2 -9 -z -c > /dev/null"),
			exec.Command("sh", "-c", "xz -9 -z -c > /dev/null"),
		}

		for _, cmd := range cmds {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				log.Fatal(err)
			}
			go func() {
				file, err := os.OpenFile(file.Name(), os.O_RDONLY, 0600)
				if err != nil {
					log.Fatal(err)
				}
				defer file.Close()
				if _, err := io.Copy(stdin, file); err != nil {
					log.Fatal(err)
				}
				if err := stdin.Close(); err != nil {
					log.Fatal(err)
				}
			}()
		}

		for _, cmd := range cmds {
			if err := cmd.Start(); err != nil {
				log.Fatal(err)
			}
		}

		for _, cmd := range cmds {
			if err := cmd.Wait(); err != nil {
				log.Fatal(err)
			}
		}

		fmt.Println("io.Pipe/io.Copy compressed 8mb in gzip, bzip2, and xz in", time.Since(start))
	}()

}
