package main

import (
	"emul8"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("must specify file")
	}
	name := os.Args[1]

	var e emul8.Emulator

	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	err = e.Load(b)
	if err != nil {
		log.Fatal(err)
	}

	err = e.Run()
	if err != nil {
		log.Fatal(err)
	}
}
