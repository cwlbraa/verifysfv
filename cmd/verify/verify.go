package main

import (
	"flag"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/cwlbraa/verifysfv"
	"github.com/gosuri/uiprogress"
)

func main() {
	poly := flag.String("poly", "crc32c", "polynomial: defaults to crc32c (Castagnoli), alternatives: ieee, koopman")
	memory := flag.Int("mem", runtime.NumCPU()*4, "kBs of memory to read files into")
	flag.Parse()
	verifysfv.SetBufSize(*memory * 1024 / runtime.NumCPU())
	polynomial := parsePoly(*poly)

	if len(flag.Args()) < 1 {
		fmt.Println("please provide an sfv file to verify")
		os.Exit(1)
	}
	sfvFilepath := flag.Args()[0]
	parsed, err := verifysfv.Read(sfvFilepath)
	if err != nil {
		log.Fatal(err)
	}

	count := len(parsed.Checksums)
	bar := uiprogress.AddBar(count).AppendCompleted().PrependElapsed()

	checksums := make(chan verifysfv.Checksum, count)
	errs := make(chan error, count) // nil errors indicate success
	var wg sync.WaitGroup

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			for checksum := range checksums {
				success, result, err := checksum.Verify(polynomial)
				bar.Incr()

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

	uiprogress.Start()
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

func parsePoly(in string) uint32 {
	switch in {
	case "crc32c":
		return crc32.Castagnoli
	case "ieee":
		return crc32.IEEE
	case "koop":
		return crc32.Koopman
	default:
		fmt.Println("unsupported polynomial")
		os.Exit(1)
	}
	return 0
}
