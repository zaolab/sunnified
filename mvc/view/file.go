package view

import (
	"bytes"
	"fmt"
	"github.com/zaolab/sunnified/web"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"time"
)

type FileDownloadView struct {
	FilePath string
	CType    string
	FileName string
}

func (this *FileDownloadView) ContentType(ctxt *web.Context) string {
	if this.CType == "" {
		this.CType = mime.TypeByExtension(path.Ext(this.FilePath))
		if this.CType == "" {
			this.CType = "application/octet-stream"
		}
	}
	return this.CType
}

func (this *FileDownloadView) Render(ctxt *web.Context) ([]byte, error) {
	var file *os.File
	var err error

	if file, err = os.Open(this.FilePath); err == nil {
		defer file.Close()
		var bsize int64 = 1000

		if stat, err := file.Stat(); err == nil {
			bsize = stat.Size()
		}
		buf := bytes.NewBuffer(make([]byte, 0, bsize))
		io.Copy(buf, file)
		return buf.Bytes(), nil
	}

	return nil, err
}

func (this *FileDownloadView) RenderString(ctxt *web.Context) (string, error) {
	b, err := this.Render(ctxt)
	if err == nil {
		return string(b), nil
	}
	return "", err
}

func (this *FileDownloadView) Publish(ctxt *web.Context) (err error) {
	var file *os.File

	if file, err = os.Open(this.FilePath); err == nil {
		defer file.Close()

		fbase := path.Base(this.FilePath)
		if this.FileName == "" {
			this.FileName = fbase
		}

		header := ctxt.Response.Header()
		header.Set("Content-Type", this.ContentType(ctxt))
		header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, this.FileName))

		var modtime time.Time
		if stat, err := file.Stat(); err == nil {
			modtime = stat.ModTime()
		}

		http.ServeContent(ctxt.Response, ctxt.Request, fbase, modtime, file)
	} else {
		ctxt.SetErrorCode(404)
	}

	return
}

func NewFileDownloadView(fpath string, fname string, ctype string) *FileDownloadView {
	return &FileDownloadView{
		FilePath: fpath,
		FileName: fname,
		CType:    ctype,
	}
}
