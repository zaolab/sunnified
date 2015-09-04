package handler

import (
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"github.com/zaolab/sunnified/util/validate"
	"github.com/zaolab/sunnified/web/resp"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const GZIP_EXT = ".gz"
const TYPE_DEFAULT = "application/octet-stream"

type StaticFileHandler struct {
	BasePath    string // relative path from application or absolute path
	BaseURL     string // relative path of domain
	DefaultType string
	Cache       int
	Gzip        []string
	GzippedFile bool
}

func NewStaticFileHandler() *StaticFileHandler {
	return &StaticFileHandler{}
}

func NewStaticFileHandlerPath(basepath string, baseurl string) *StaticFileHandler {
	return &StaticFileHandler{
		BasePath: basepath,
		BaseURL:  baseurl,
	}
}

func (this *StaticFileHandler) ServeOptions(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
	w.Header().Set("Allow", "HEAD GET OPTIONS")
}

func (this *StaticFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		gzipextl = len(this.Gzip)
		urlpath  = strings.Trim(r.URL.Path, "/")
		basepath = this.BasePath
		header   = w.Header()
		fullpath string
		file     *os.File
		err      error
		etaggz   string
	)

	if urlpath == "" || strings.Contains(urlpath, "../") {
		NotFound(w, r)
		return
	}

	if basepath != "" && basepath[len(basepath)-1] != '/' {
		basepath = basepath + "/"
	}

	fullpath = basepath + urlpath

	if this.BaseURL != "" {
		if !strings.HasPrefix(urlpath, this.BaseURL) {
			NotFound(w, r)
			return
		}

		urlpath = strings.Trim(urlpath[len(this.BaseURL):], "/")
	}

	st, err := os.Stat(fullpath)

	if err == nil {
		if st.IsDir() {
			fullpath = fullpath + "/index.html"
			st, err = os.Stat(fullpath)

			if err != nil || st.IsDir() {
				NotFound(w, r)
				return
			}
		}
	} else {
		NotFound(w, r)
		return
	}

	var clen int64 = st.Size()
	var modtime time.Time = st.ModTime()
	var ext = path.Ext(fullpath)
	var usegzip = (gzipextl == 0 && this.GzippedFile) || (gzipextl > 0 && (this.Gzip[0] == "*" || validate.IsIn(ext, this.Gzip...)))

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && usegzip {
		if this.GzippedFile {
			var gzpath = fullpath + GZIP_EXT
			var stgz, err = os.Stat(gzpath)

			if err == nil && !stgz.IsDir() && (stgz.ModTime().After(modtime) || stgz.ModTime().Equal(modtime)) {
				modtime = stgz.ModTime()
				ctype := mime.TypeByExtension(ext)

				if ctype == "" {
					if this.DefaultType == "" {
						ctype = TYPE_DEFAULT
					} else {
						ctype = this.DefaultType
					}
				}

				clen = stgz.Size()
				header.Set("Content-Type", ctype)

				if r.Header.Get("Range") == "" {
					header.Set("Content-Length", strconv.FormatInt(clen, 10))
				}

				fullpath = gzpath
				usegzip = false
			}
		}

		etaggz = GZIP_EXT
		header.Set("Content-Encoding", "gzip")
	}

	md5ed := md5.Sum([]byte(fmt.Sprintf("%d%s%d", clen, fullpath, modtime.Unix())))
	etag := fmt.Sprintf("%x%s", md5ed, etaggz)
	header.Set("ETag", etag)

	file, err = os.Open(fullpath)

	if err != nil {
		NotFound(w, r)
		return
	}

	defer file.Close()

	if this.Cache != 0 {
		header.Set("Cache-Control", fmt.Sprintf("max-age=%d", this.Cache))
	} else {
		header.Set("Cache-Control", "max-age=0, must-revalidate")
	}

	if usegzip {
		var gzfile *os.File

		// serveContent will not write to response if client already has a copy of file making the .gz local file empty
		if this.GzippedFile && r.Method != "HEAD" && r.Header.Get("If-Modified-Since") == "" && r.Header.Get("If-None-Match") == "" && r.Header.Get("If-Range") == "" {
			gzfile, err = os.Create(fullpath + GZIP_EXT)
			if err == nil {
				defer gzfile.Close()
			}
		}

		// TODO: this will probably mess up with clients
		// who request range since the original size and zipped size is diff
		// we can jus do our own io.Copy without support for range
		gzw := resp.NewGzipResponseWriterLevelFile(w, r, gzip.BestSpeed, gzfile)
		defer gzw.Close()
		w = gzw
	} else {
		// the gzipresponsewriter will add its own accept-encoding
		header.Add("Vary", "Accept-Encoding")
	}

	http.ServeContent(w, r, path.Base(urlpath), modtime, file)
}
