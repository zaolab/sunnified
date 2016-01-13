package av

import (
	"errors"
	"io"
)

var ErrInvalidFileName = errors.New("Invalid filename")

type AVScanner interface {
	ScanFile(string) (AVResult, error)
	ScanBytes([]byte) (AVResult, error)
	ScanStream(io.Reader) (AVResult, error)
	ScanFileAsync(string) <-chan AVResultErr
	ScanBytesAsync([]byte) <-chan AVResultErr
	ScanStreamAsync(io.Reader) <-chan AVResultErr
}

type AVResult struct {
	FileName string
	Status   bool
	Virus    string
}

type AVResultErr struct {
	AVResult
	Error error
}
