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
}

func (gw *GzipResponseWriter) Write(data []byte) (n int, err error) {
	if gw.gzip != nil {
		defer gw.gzip.Flush()
		return gw.gzip.Write(data)
	}
	return gw.ResponseWriter.Write(data)
}

// GzipResponseWriter must be manually closed!
func (gw *GzipResponseWriter) Close() {
	if gw.gzip != nil {
		gw.gzip.Close()
	}
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
		}

		header := w.Header()
		header.Set("Content-Encoding", "gzip")
		if vary, exists := header["Vary"]; !exists || !validate.IsIn("Accept-Encoding", vary...) {
			header.Add("Vary", "Accept-Encoding")
		}

		return resp
	}

	return &GzipResponseWriter{w, nil}
}

func NewGzipResponseWriterLevel(w http.ResponseWriter, r *http.Request, level int) *GzipResponseWriter {
	return NewGzipResponseWriterLevelFile(w, r, level, nil)
}

func NewGzipResponseWriter(w http.ResponseWriter, r *http.Request) *GzipResponseWriter {
	return NewGzipResponseWriterLevel(w, r, gzip.DefaultCompression)
}
