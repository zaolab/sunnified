package av

import (
	"errors"
	"io"
)

var ErrInvalidFileName = errors.New("Invalid filename")

type Scanner interface {
	ScanFile(string) (Result, error)
	ScanBytes([]byte) (Result, error)
	ScanStream(io.Reader) (Result, error)
	ScanFileAsync(string) <-chan ResultErr
	ScanBytesAsync([]byte) <-chan ResultErr
	ScanStreamAsync(io.Reader) <-chan ResultErr
}

type Result struct {
	FileName string
	Status   bool
	Virus    string
}

type ResultErr struct {
	Result
	Error error
}
