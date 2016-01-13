package av

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
)

var (
	clamCmdInstream        = []byte("zINSTREAM\x00")
	clamCmdScan            = "zSCAN %s\x00"
	clamInstreamBufSize    = 64 * 1024
	clamInstreamBufSizeBin = make([]byte, 4)
)

func init() {
	binary.BigEndian.PutUint32(clamInstreamBufSizeBin, uint32(clamInstreamBufSize))
}

func NewClamAVScanner(network, address string) AVScanner {
	return ClamAVScanner{
		network: network,
		address: address,
	}
}

type ClamAVScanner struct {
	network string
	address string
}

func (this ClamAVScanner) ScanFile(filename string) (res AVResult, err error) {
	var (
		conn    net.Conn
		absname string
	)

	if conn, err = net.Dial(this.network, this.address); err != nil {
		return
	}
	defer conn.Close()

	if hasInvalidChars(filename) {
		err = ErrInvalidFileName
		return
	}
	if absname, err = filepath.Abs(filename); err != nil {
		return
	}

	if _, err = conn.Write([]byte(fmt.Sprintf(clamCmdScan, absname))); err != nil {
		return
	}

	res, err = parseResult(conn)
	res.FileName = filename

	return
}

func (this ClamAVScanner) ScanBytes(d []byte) (res AVResult, err error) {
	var (
		conn net.Conn
		ld   int
	)

	if conn, err = net.Dial(this.network, this.address); err != nil {
		return
	}
	defer conn.Close()

	if _, err = conn.Write(clamCmdInstream); err != nil {
		return
	}

	start := 0
	end := 0

	if ld = len(d); ld >= clamInstreamBufSize {
		rounds := ld / clamInstreamBufSize

		for i := 0; i < rounds; i++ {
			end += clamInstreamBufSize
			conn.Write(clamInstreamBufSizeBin)
			conn.Write(d[start:end])
			start = end
		}
	}

	if end != ld {
		end = ld
		intbyte := make([]byte, 4)
		binary.BigEndian.PutUint32(intbyte, uint32(end-start))
		conn.Write(intbyte)
		conn.Write(d[start:end])
	}

	conn.Write([]byte{0, 0, 0, 0})

	res, err = parseResult(conn)
	return
}

func (this ClamAVScanner) ScanStream(r io.Reader) (res AVResult, err error) {
	var (
		conn net.Conn
		buff = make([]byte, clamInstreamBufSize)
	)

	if conn, err = net.Dial(this.network, this.address); err != nil {
		return
	}
	defer conn.Close()

	if _, err = conn.Write(clamCmdInstream); err != nil {
		return
	}

	for i, e := r.Read(buff); i > 0; i, e = r.Read(buff) {
		if i == clamInstreamBufSize {
			conn.Write(clamInstreamBufSizeBin)
		} else {
			intbyte := make([]byte, 4)
			binary.BigEndian.PutUint32(intbyte, uint32(i))
			conn.Write(intbyte)
		}

		conn.Write(buff[0:i])

		if e != nil {
			break
		}
	}

	conn.Write([]byte{0, 0, 0, 0})

	res, err = parseResult(conn)
	return
}

func (this ClamAVScanner) ScanFileAsync(filename string) (c <-chan AVResultErr) {
	c = make(chan AVResultErr, 1)

	go func() {
		res, err := this.ScanFile(filename)
		c <- AVResultErr{
			AVResult: res,
			Error:    err,
		}
	}()

	return c
}

func (this ClamAVScanner) ScanBytesAsync(d []byte) (c <-chan AVResultErr) {
	c = make(chan AVResultErr, 1)

	go func() {
		res, err := this.ScanBytes(d)
		c <- AVResultErr{
			AVResult: res,
			Error:    err,
		}
	}()

	return c
}

func (this ClamAVScanner) ScanStreamAsync(r io.Reader) (c <-chan AVResultErr) {
	c = make(chan AVResultErr, 1)

	go func() {
		res, err := this.ScanStream(r)
		c <- AVResultErr{
			AVResult: res,
			Error:    err,
		}
	}()

	return c
}

func hasInvalidChars(filename string) bool {
	return strings.ContainsAny(filename, "*?\"<>|\r\n\x00")
}

func parseResult(conn net.Conn) (res AVResult, err error) {
	var val []byte

	if val, err = ioutil.ReadAll(conn); err != nil {
		return
	}

	status := strings.SplitN(strings.Trim(string(val), "\x00"), ":", 2)
	stval := strings.TrimSpace(status[1])

	if strings.HasSuffix(stval, "FOUND") {
		res.Virus = strings.TrimSpace(strings.TrimSuffix(stval, "FOUND"))
	} else if strings.HasSuffix(stval, "ERROR") {
		err = errors.New(strings.TrimSpace(strings.TrimSuffix(stval, "ERROR")))
	} else {
		res.Status = true
	}

	return
}
