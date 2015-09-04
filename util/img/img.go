package img

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/nfnt/resize"
	"github.com/zaolab/sunnified/util"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path"
	"strings"
)

var ErrImageExceedSize = errors.New("Image size too large")

func SaveThumbnailAndImage(imgr io.ReadSeeker, maxw, maxh int, tdir, dir, fname string) (tname, name string, err error) {
	var resized bool
	tname, resized, err = saveThumbnail(imgr, maxw, maxh, tdir, fname)
	if err != nil {
		return
	}

	var source io.Reader
	var file *os.File

	if resized {
		imgr.Seek(0, 0)
		var s224 = sha256.New224()
		if _, err = io.Copy(s224, imgr); err != nil {
			return
		}

		hash := make([]byte, 0, sha256.Size224)
		hash = s224.Sum(hash)
		name = fmt.Sprintf("%x%s", hash, path.Ext(tname))
		imgr.Seek(0, 0)

		if _, err = os.Stat(dir + name); err == nil {
			return
		} else if !os.IsNotExist(err) {
			return
		}
		source = imgr
	} else {
		var sourcefile *os.File
		if sourcefile, err = os.Open(util.AddDirTrailSlash(tdir) + tname); err != nil {
			return
		}
		defer sourcefile.Close()
		source = sourcefile
		name = tname
	}

	if file, err = os.Create(dir + name); err != nil {
		return
	}
	defer file.Close()

	_, err = io.Copy(file, source)
	return
}

func saveThumbnail(imgr io.ReadSeeker, maxw, maxh int, dir, fname string) (name string, resized bool, err error) {
	dir = util.AddDirTrailSlash(dir)
	fname = strings.TrimSpace(fname)

	var (
		img  image.Image
		imgf string
		imgc image.Config
		file *os.File
		rdr  io.Reader = imgr
	)

	if imgc, imgf, err = image.DecodeConfig(imgr); err != nil {
		return
	}

	if imgf == "jpeg" {
		imgf = "jpg"
	}

	if imgc.Width*imgc.Height > 100000000 {
		// image is bigger than 100megapixels
		err = ErrImageExceedSize
		return
	} else if imgc.Width > maxw || imgc.Height > maxh {
		imgr.Seek(0, 0)
		if img, _, err = image.Decode(imgr); err != nil {
			return
		}

		img = resize.Thumbnail(uint(maxw), uint(maxh), img, resize.Bicubic)
		resized = true
	}

	if fname == "" {
		var s224 = sha256.New224()

		if img != nil {
			buf := &bytes.Buffer{}
			writer := io.MultiWriter(s224, buf)

			if err = EncodeImg(writer, img, imgf); err != nil {
				return
			}

			img = nil
			rdr = buf
		} else {
			if _, err = io.Copy(s224, imgr); err != nil {
				return
			}

			imgr.Seek(0, 0)
		}

		hash := make([]byte, 0, sha256.Size224)
		hash = s224.Sum(hash)
		name = fmt.Sprintf("%x.%s", hash, imgf)

		if _, err = os.Stat(dir + name); err == nil {
			return
		} else if !os.IsNotExist(err) {
			return
		}
	} else {
		name = fname + "." + imgf

		if _, err = os.Stat(dir + name); err == nil {
			err = os.ErrExist
			return
		} else if !os.IsNotExist(err) {
			return
		}

	}

	if file, err = os.Create(dir + name); err != nil {
		return
	}
	defer file.Close()

	if img != nil {
		EncodeImg(file, img, imgf)
	} else {
		_, err = io.Copy(file, rdr)
	}

	return
}

func SaveThumbnail(imgr io.ReadSeeker, maxw, maxh int, dir, fname string) (name string, err error) {
	name, _, err = saveThumbnail(imgr, maxw, maxh, dir, fname)
	return
}

func EncodeImg(writer io.Writer, img image.Image, imgf string) (err error) {
	imgf = strings.ToLower(imgf)

	switch imgf {
	case "gif":
		err = gif.Encode(writer, img, nil)
	case "png":
		err = png.Encode(writer, img)
	default:
		err = jpeg.Encode(writer, img, &jpeg.Options{Quality: 90})
	}

	return
}
