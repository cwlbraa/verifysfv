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

// command line option configuration
var poly = flag.String("poly", "crc32c", "crc base polynomial: crc32c (Castagnoli), ieee, or koopman")
var parallelism = flag.Int("j", runtime.NumCPU(), "# of parallel workers to spin up")
var memory = flag.Int("mem", runtime.NumCPU()*4, "kBs of memory to use as file buffers")

func main() {
	flag.Usage = func() {
		fmt.Printf("verify: a tiny, fast, io-bound tool for verifying sfv files\n\n")
		fmt.Printf("Usage: verify [options] fileManifest.sfv\n\n")
		fmt.Printf("options:\n")
		flag.PrintDefaults()
	}
	// parse and verify args
	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	sfvFilepath := flag.Args()[0]
	polynomial := parsePoly(*poly)
	verifysfv.SetBufSize(*memory * 1024 / *parallelism)

	// open and parse sfv file
	parsed, err := verifysfv.Read(sfvFilepath)
	if err != nil {
		log.Fatal(err)
	}

	count := len(parsed.Checksums)
	bar := uiprogress.AddBar(count).AppendCompleted().PrependElapsed()

	checksums := make(chan verifysfv.Checksum, count)
	errs := make(chan error, count) // nil errors indicate success
	var wg sync.WaitGroup

	for i := 0; i < *parallelism; i++ {
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
		log.Fatalf("unsupported polynomial %s", in)
	}
	return 0
}
