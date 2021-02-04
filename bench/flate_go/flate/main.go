package main

import (
	"compress/gzip"
	"io"
	"log"
	"os"
)

func main() {
	rp, wp := io.Pipe()

	writer := gzip.NewWriter(wp)
	go func() {
		if _, err := io.Copy(writer, os.Stdin); err != nil {
			log.Fatal(err)
		}
		if err := writer.Flush(); err != nil {
			log.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			log.Fatal(err)
		}
		if err := wp.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	reader, err := gzip.NewReader(rp)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = io.Copy(os.Stdout, reader); err != nil {
		log.Fatal(err)
	}
}
