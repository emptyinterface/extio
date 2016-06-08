package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"

	"github.com/emptyinterface/extio"
)

func main() {

	var words []string

	cmd := exec.Command("cat", "/usr/share/dict/words")
	cmd.Stdout = extio.NewScannerWriter(bufio.ScanWords, 1<<10, func(token []byte) error {
		words = append(words, string(token))
		return nil
	})

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("extio.ScannerWriter scanned", len(words), "words")

}
