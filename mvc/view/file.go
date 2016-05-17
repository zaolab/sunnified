package view

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/zaolab/sunnified/web"
)

type FileDownloadView struct {
	FilePath string
	CType    string
	FileName string
}

func (fv *FileDownloadView) ContentType(ctxt *web.Context) string {
	if fv.CType == "" {
		fv.CType = mime.TypeByExtension(path.Ext(fv.FilePath))
		if fv.CType == "" {
			fv.CType = "application/octet-stream"
		}
	}
	return fv.CType
}

func (fv *FileDownloadView) Render(ctxt *web.Context) ([]byte, error) {
	var file *os.File
	var err error

	if file, err = os.Open(fv.FilePath); err == nil {
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

func (fv *FileDownloadView) RenderString(ctxt *web.Context) (string, error) {
	b, err := fv.Render(ctxt)
	if err == nil {
		return string(b), nil
	}
	return "", err
}

func (fv *FileDownloadView) Publish(ctxt *web.Context) (err error) {
	var file *os.File

	if file, err = os.Open(fv.FilePath); err == nil {
		defer file.Close()

		fbase := path.Base(fv.FilePath)
		if fv.FileName == "" {
			fv.FileName = fbase
		}

		header := ctxt.Response.Header()
		header.Set("Content-Type", fv.ContentType(ctxt))
		header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fv.FileName))

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
