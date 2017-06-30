// Package sfv provides a simple way of reading and verifying SFV (Simple File
// Verification) files.
package verifysfv

import (
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

// Checksum represents a line in a SFV file, containing the filename, full path
// to the file and the CRC32 checksum
type Checksum struct {
	Filename string
	Path     string
	CRC32    uint32
}

// SFV contains all the checksums read from a SFV file.
type SFV struct {
	Checksums []Checksum
	Path      string
}

var bufSize uint64 = 4096

func SetBufSize(bs int) {
	atomic.SwapUint64(&bufSize, uint64(bs))
}

func GetBufSize() uint64 {
	return atomic.LoadUint64(&bufSize)
}

// Verify calculates the CRC32 of the associated file and returns true if the
// checksum is correct along with the calculated checksum
func (c *Checksum) Verify(polynomial uint32) (bool, uint32, error) {
	f, err := os.Open(c.Path)
	if err != nil {
		return false, 0, err
	}
	defer f.Close()

	h := crc32.New(crc32.MakeTable(polynomial))
	reader := bufio.NewReader(f)
	buf := make([]byte, bufSize)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return false, 0, err
		}
		if n == 0 {
			break
		}
		h.Write(buf[:n])
	}
	result := h.Sum32()

	return result == c.CRC32, result, nil
}

// IsExist returns a boolean indicating if the file associated with the checksum
// exists
func (c *Checksum) IsExist() bool {
	_, err := os.Stat(c.Path)
	return err == nil
}

// Verify verifies all checksums contained in SFV and returns true if all
// checksums are correct.
func (s *SFV) Verify(polynomial uint32) (bool, error) {
	if len(s.Checksums) == 0 {
		return false, fmt.Errorf("no checksums found in %s", s.Path)
	}
	for _, c := range s.Checksums {
		ok, _, err := c.Verify(polynomial)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// IsExist returns a boolean if all the files in SFV exists
func (s *SFV) IsExist() bool {
	for _, c := range s.Checksums {
		if !c.IsExist() {
			return false
		}
	}
	return true
}

func parseChecksum(dir string, line string) (*Checksum, error) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("could not parse checksum: %q", line)
	}
	filename := strings.TrimSpace(parts[0])
	path := path.Join(dir, filename)
	crc32, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 16, 32)
	if err != nil {
		return nil, err
	}
	// ParseUint will return error if number exceeds 32 bits
	return &Checksum{
		Path:     path,
		Filename: filename,
		CRC32:    uint32(crc32),
	}, nil
}

func parseChecksums(dir string, r io.Reader) ([]Checksum, error) {
	checksums := []Checksum{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, ";") {
			continue
		}
		checksum, err := parseChecksum(dir, line)
		if err != nil {
			return nil, err
		}
		checksums = append(checksums, *checksum)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return checksums, nil
}

// Read reads a SFV file from filepath and creates a new SFV containing
// checksums parsed from the SFV file.
func Read(filepath string) (*SFV, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dir := path.Dir(filepath)
	checksums, err := parseChecksums(dir, f)
	if err != nil {
		return nil, err
	}
	return &SFV{
		Checksums: checksums,
		Path:      filepath,
	}, nil
}

// Find tries to find a SFV file in the given path. If multiple SFV files exist
// in path, the first one will be returned.
func Find(path string) (*SFV, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sfv" {
			return Read(filepath.Join(path, f.Name()))
		}
	}
	return nil, fmt.Errorf("no sfv found in %s", path)
}
