package main

import (
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"runtime"
	"sync"

	pb "gopkg.in/cheggaaa/pb.v2"

	"github.com/mpolden/sfv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("please provide an sfv file to verify")
	}
	sfvFilepath := os.Args[1]
	parsed, err := sfv.Read(sfvFilepath)
	if err != nil {
		log.Fatal(err)
	}

	count := len(parsed.Checksums)
	bar := pb.StartNew(count)
	checksums := make(chan sfv.Checksum, count)
	errs := make(chan error, count) // nil errors indicate success
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			for checksum := range checksums {
				success, result, err := checksum.Verify(crc32.Castagnoli)
				bar.Increment()

				if !success && err == nil {
					errs <- fmt.Errorf("corruption: expected %x but computed %x for %s\n",
						checksum.CRC32, result, checksum.Filename)
					continue
				}

				errs <- err // nil error indicates success
			}
			wg.Done()
		}()
	}

	for _, chk := range parsed.Checksums {
		checksums <- chk
	}
	close(checksums)

	// close errs asyncronously so we can print errors as we get them
	go func() {
		wg.Wait()
		close(errs)
	}()

	exitCode := 0
	for err := range errs {
		if err != nil {
			exitCode = 1
			fmt.Println(err)
		}
	}

	os.Exit(exitCode)
}
