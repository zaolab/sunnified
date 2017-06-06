package resp

import (
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/zaolab/sunnified/util/validate"
)

type GzipResponseWriter struct {
	http.ResponseWriter
	gzip *gzip.Writer
	size int64
}

func (gw *GzipResponseWriter) Write(data []byte) (int, error) {
	if gw.gzip != nil {
		var (
			i   int
			err error
		)

		if i, err = gw.gzip.Write(data); err == nil {
			if err = gw.gzip.Flush(); err == nil {
				gw.size += int64(i)
			} else {
				i = 0
			}
		}

		return i, err
	}

	return gw.ResponseWriter.Write(data)
}

// GzipResponseWriter must be manually closed!
func (gw *GzipResponseWriter) Close() error {
	if gw.gzip != nil {
		return gw.gzip.Close()
	}

	return nil
}

func (gw *GzipResponseWriter) RawSize() int64 {
	return gw.size
}

func NewGzipResponseWriterLevelFile(w http.ResponseWriter, r *http.Request, level int, file *os.File) *GzipResponseWriter {
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		if level < gzip.DefaultCompression || level > gzip.BestCompression {
			level = gzip.DefaultCompression
		}

		var gz *gzip.Writer

		if file != nil {
			gz, _ = gzip.NewWriterLevel(io.MultiWriter(w, file), level)
		} else {
			gz, _ = gzip.NewWriterLevel(w, level)
		}

		resp := &GzipResponseWriter{
			ResponseWriter: w,
			gzip:           gz,
			size:           0,
		}

		header := w.Header()
		header.Set("Content-Encoding", "gzip")
		if vary, exists := header["Vary"]; !exists || !validate.IsIn("Accept-Encoding", vary...) {
			header.Add("Vary", "Accept-Encoding")
		}

		return resp
	}

	return &GzipResponseWriter{w, nil, 0}
}

func NewGzipResponseWriterLevel(w http.ResponseWriter, r *http.Request, level int) *GzipResponseWriter {
	return NewGzipResponseWriterLevelFile(w, r, level, nil)
}

func NewGzipResponseWriter(w http.ResponseWriter, r *http.Request) *GzipResponseWriter {
	return NewGzipResponseWriterLevel(w, r, gzip.DefaultCompression)
}
