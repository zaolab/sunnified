package web

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/zaolab/sunnified/util/validate"
)

type UPath []string
type PData map[string]string

type StatusCode int

type PathString string
type PathInt int
type PathInt64 int64
type PathFloat float32
type PathFloat64 float64

type FormString string
type FormInt int
type FormInt64 int64
type FormFloat float32
type FormFloat64 float64
type FormEmail string
type FormURL string
type FormDate time.Time
type FormTime time.Duration
type FormDateTime time.Time

func (upath UPath) String(index int) (s string, err error) {
	if index >= 0 && len(upath) > index {
		s = upath[index]
	} else {
		err = errors.New("index of out range")
	}
	return
}

func (upath UPath) Int(index int) (i int, err error) {
	var s string
	if s, err = upath.String(index); err == nil {
		i, err = strconv.Atoi(s)
	}
	return
}

func (upath UPath) Int64(index int) (i int64, err error) {
	var s string
	if s, err = upath.String(index); err == nil {
		i, err = strconv.ParseInt(s, 10, 0)
	}
	return
}

func (upath UPath) Float32(index int) (i float32, err error) {
	var s string
	if s, err = upath.String(index); err == nil {
		var f64 float64
		f64, err = strconv.ParseFloat(s, 32)
		i = float32(f64)
	}
	return
}

func (upath UPath) Float64(index int) (i float64, err error) {
	var s string
	if s, err = upath.String(index); err == nil {
		i, err = strconv.ParseFloat(s, 64)
	}
	return
}

func (upath UPath) GetString(index int, def ...string) (s string) {
	if index >= 0 && len(upath) > index {
		s = upath[index]
	} else if len(def) > 0 {
		s = def[0]
	}
	return
}

func (upath UPath) GetInt(index int, def ...int) (i int) {
	var s string
	var err error
	if s, err = upath.String(index); err == nil {
		i, err = strconv.Atoi(s)
	}
	if err != nil && len(def) > 0 {
		i = def[0]
	}
	return
}

func (upath UPath) GetInt64(index int, def ...int64) (i int64) {
	var s string
	var err error
	if s, err = upath.String(index); err == nil {
		i, err = strconv.ParseInt(s, 10, 0)
	}
	if err != nil && len(def) > 0 {
		i = def[0]
	}
	return
}

func (upath UPath) GetFloat32(index int, def ...float32) (i float32) {
	var s string
	var err error
	if s, err = upath.String(index); err == nil {
		var f64 float64
		f64, err = strconv.ParseFloat(s, 32)
		i = float32(f64)
	}
	if err != nil && len(def) > 0 {
		i = def[0]
	}
	return
}

func (upath UPath) GetFloat64(index int, def ...float64) (i float64) {
	var s string
	var err error
	if s, err = upath.String(index); err == nil {
		i, err = strconv.ParseFloat(s, 64)
	}
	if err != nil && len(def) > 0 {
		i = def[0]
	}
	return
}

func (data PData) String(key string) (s string, err error) {
	if slice, ok := data[key]; ok {
		s = slice
	} else {
		err = errors.New("invalid key")
	}
	return
}
func (data PData) Int(key string) (i int, err error) {
	var s string
	if s, err = data.String(key); err == nil {
		i, err = strconv.Atoi(s)
	}
	return
}

func (data PData) Int64(key string) (i int64, err error) {
	var s string
	if s, err = data.String(key); err == nil {
		i, err = strconv.ParseInt(s, 10, 0)
	}
	return
}

func (data PData) Float32(key string) (i float32, err error) {
	var s string
	if s, err = data.String(key); err == nil {
		var f64 float64
		f64, err = strconv.ParseFloat(s, 32)
		i = float32(f64)
	}
	return
}

func (data PData) Float64(key string) (i float64, err error) {
	var s string
	if s, err = data.String(key); err == nil {
		i, err = strconv.ParseFloat(s, 64)
	}
	return
}

func (data PData) Email(key string) (s string, err error) {
	if s, err = data.String(key); err == nil {
		if !validate.IsEmail(s) {
			err = errors.New("invalid email")
		}
	}
	return
}

func (data PData) Url(key string) (string, error) {
	return data.URL(key)
}

func (data PData) URL(key string) (s string, err error) {
	if s, err = data.String(key); err == nil {
		if !validate.IsURL(s) {
			err = errors.New("invalid url")
		}
	}
	return
}

func (data PData) Date(key string) (d time.Time, err error) {
	var s string
	if s, err = data.String(key); err == nil {
		d, err = time.Parse("2013-02-13", s)
	}
	return
}

func (data PData) Time(key string) (t time.Duration, err error) {
	var s string

	if s, err = data.String(key); err == nil {
		if strings.Index(s, ":") >= 0 {
			s = strings.Replace(s, ":", "h", 1)
			if strings.Index(s, ":") >= 0 {
				s = strings.Replace(s, ":", "m", 1)
				s = s + "s"
			} else {
				s = s + "m"
			}
		}

		t, err = time.ParseDuration(s)
	}

	return
}

func (data PData) DateTime(key string) (dt time.Time, err error) {
	var s string

	if s, err = data.String(key); err == nil {
		dt, err = time.Parse("2013-02-13T03:04:05+0000", s)
	}

	return
}
