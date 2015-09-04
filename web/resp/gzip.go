package resp

import (
	"compress/gzip"
	"github.com/zaolab/sunnified/util/validate"
	"io"
	"net/http"
	"os"
	"strings"
)

type GzipResponseWriter struct {
	http.ResponseWriter
	gzip *gzip.Writer
}

func (this *GzipResponseWriter) Write(data []byte) (n int, err error) {
	if this.gzip != nil {
		defer this.gzip.Flush()
		return this.gzip.Write(data)
	}
	return this.ResponseWriter.Write(data)
}

// GzipResponseWriter must be manually closed!
func (this *GzipResponseWriter) Close() {
	if this.gzip != nil {
		this.gzip.Close()
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
