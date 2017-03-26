package util

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrCastFailed = errors.New("type casting failed")

func FileExists(f string) bool {
	_, err := os.Stat(f)
	return err == nil
}

func StringSplitLastN(s, sep string, n int) (splitted []string) {
	if n == 0 {
		return
	} else if n == 1 {
		return []string{s}
	} else if n < 0 {
		return strings.Split(s, sep)
	}

	var lens = len(s)

	if sep == "" {
		if l := utf8.RuneCountInString(s); n > l {
			splitted = strings.Split(s, sep)
		} else {
			splitted = make([]string, n)
			end := lens

			for i := n - 1; i >= 1; i-- {
				c, size := utf8.DecodeLastRuneInString(s[:end])
				if c == utf8.RuneError {
					splitted[i] = string(utf8.RuneError)
				} else {
					splitted[i] = s[end-size : end]
				}
				end = end - size
			}

			splitted[0] = s[:end]
		}
	} else {
		splitted = make([]string, n)
		var (
			sindex = n - 1
			lensep = len(sep)
			lasti  = lens
		)

		for i := lens - lensep; i >= 0 && sindex > 0; i-- {
			if s[i:i+lensep] == sep {
				splitted[sindex] = s[i+lensep : lasti]
				sindex--
				lasti = i
				i = i - lensep + 1
			}
		}

		splitted[sindex] = s[0:lasti]
		if sindex != 0 {
			splitted = splitted[sindex:]
		}
	}

	return splitted
}

func CastInt64(value interface{}) (def int64, err error) {
	switch val := value.(type) {
	case int64:
		def = val
	case string:
		v, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			def = v
		}
	case json.Number:
		v, err := val.Int64()
		if err == nil {
			def = v
		}
	case int:
		def = int64(val)
	case int8:
		def = int64(val)
	case int16:
		def = int64(val)
	case int32:
		def = int64(val)
	case uint:
		def = int64(val)
	case uint8:
		def = int64(val)
	case uint16:
		def = int64(val)
	case uint32:
		def = int64(val)
	case uint64:
		def = int64(val)
	case float32:
		def = int64(val)
	case float64:
		def = int64(val)
	default:
		err = ErrCastFailed
	}

	return
}

func CastUint64(value interface{}) (def uint64, err error) {
	switch val := value.(type) {
	case uint64:
		def = val
	case string:
		v, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			def = uint64(v)
		}
	case json.Number:
		v, err := val.Int64()
		if err == nil {
			def = uint64(v)
		}
	case int:
		def = uint64(val)
	case int8:
		def = uint64(val)
	case int16:
		def = uint64(val)
	case int32:
		def = uint64(val)
	case int64:
		def = uint64(val)
	case uint:
		def = uint64(val)
	case uint8:
		def = uint64(val)
	case uint16:
		def = uint64(val)
	case uint32:
		def = uint64(val)
	case float32:
		def = uint64(val)
	case float64:
		def = uint64(val)
	default:
		err = ErrCastFailed
	}

	return
}

func CastFloat64(value interface{}) (def float64, err error) {
	switch val := value.(type) {
	case float64:
		def = val
	case string:
		v, err := strconv.ParseFloat(val, 64)
		if err == nil {
			def = v
		}
	case json.Number:
		v, err := val.Float64()
		if err == nil {
			def = v
		}
	case int:
		def = float64(val)
	case int8:
		def = float64(val)
	case int16:
		def = float64(val)
	case int32:
		def = float64(val)
	case int64:
		def = float64(val)
	case uint:
		def = float64(val)
	case uint8:
		def = float64(val)
	case uint16:
		def = float64(val)
	case uint32:
		def = float64(val)
	case uint64:
		def = float64(val)
	case float32:
		def = float64(val)
	default:
		err = ErrCastFailed
	}

	return
}

func Must(function interface{}, args ...interface{}) interface{} {
	fVal := reflect.ValueOf(function)
	argsVal := make([]reflect.Value, len(args))

	for i, arg := range args {
		argsVal[i] = reflect.ValueOf(arg)
	}

	out := fVal.Call(argsVal)

	if len(out) == 2 {
		if out[1].Kind() == reflect.Bool {
			if !out[1].Bool() {
				goto failed
			}
		} else if out[1].Type().Implements(reflect.TypeOf(((*error)(nil))).Elem()) {
			if !out[1].IsNil() {
				goto failed
			}
		}
	}

	return out[0]

failed:
	panic("Function failed: " + fVal.Type().Name())
}

// usage MapValue(&dest, origin)
// *dest = origin
func MapValue(dest interface{}, origin interface{}) (err error) {
	defer func() {
		tmperr := recover()
		err, _ = tmperr.(error)
	}()
	destval := reflect.ValueOf(dest).Elem()
	destval.Set(reflect.ValueOf(origin))
	return
}

func MapValues(dest []interface{}, origin interface{}) (errs []error) {
	errs = make([]error, 0)
	for _, dt := range dest {
		if err := MapValue(dt, origin); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		errs = nil
	}
	return
}

func AppendValue(dest interface{}, value ...interface{}) (err error) {
	defer func() {
		tmperr := recover()
		err, _ = tmperr.(error)
	}()
	if lenval := len(value); lenval > 0 {
		sliceval := reflect.ValueOf(dest).Elem()

		newslice := reflect.MakeSlice(sliceval.Type(), 0, lenval)

		if lenval == 1 {
			reflect.Append(newslice, reflect.ValueOf(value[0]).Elem())
		} else {
			reflect.AppendSlice(newslice, reflect.ValueOf(value))
		}

		/*
			for _, val := range value {
				reflect.Append(newslice, reflect.ValueOf(val).Elem())
			}
		*/

		sliceval.Set(newslice)
	}

	return
}

type ValueGetter func(key interface{}) (interface{}, bool)

func (vg ValueGetter) Get(key interface{}, def ...interface{}) (res interface{}) {
	if val, ok := vg(key); ok {
		res = val
	} else if len(def) > 0 {
		res = def[0]
	}

	return
}

func (vg ValueGetter) GetSlice(key string, def ...[]interface{}) (res []interface{}) {
	if len(def) > 0 {
		res = def[0]
	}

	if v, ok := vg.Get(key).([]interface{}); ok {
		res = v
	}

	return
}

func (vg ValueGetter) GetBool(key string, def ...bool) (res bool) {
	if len(def) > 0 {
		res = def[0]
	}

	if v, ok := vg.Get(key).(bool); ok {
		res = v
	}

	return
}

func (vg ValueGetter) GetBoolSlice(key string, def ...[]bool) (res []bool) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []bool:
		res = v
	case []interface{}:
		sl := make([]bool, len(v))
		var ok bool
		for i, val := range v {
			if sl[i], ok = val.(bool); !ok {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetString(key string, def ...string) (res string) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case string:
		res = v
	case fmt.Stringer:
		res = v.String()
	}

	return
}

func (vg ValueGetter) GetStringSlice(key string, def ...[]string) (res []string) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []string:
		res = v
	case []interface{}:
		sl := make([]string, len(v))
		for i, val := range v {
			switch str := val.(type) {
			case string:
				sl[i] = str
			case fmt.Stringer:
				sl[i] = str.String()
			default:
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetByte(key string, def ...byte) (res byte) {
	if len(def) > 0 {
		res = def[0]
	}

	if v, ok := vg.Get(key).(byte); ok {
		res = v
	}

	return
}

func (vg ValueGetter) GetByteSlice(key string, def ...[]byte) (res []byte) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []byte:
		res = v
	case []interface{}:
		sl := make([]byte, len(v))
		var ok bool
		for i, val := range v {
			if sl[i], ok = val.(byte); !ok {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetBytes(key string, def ...[]byte) (res []byte) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []byte:
		res = v
	case string:
		res = []byte(v)
	case fmt.Stringer:
		res = []byte(v.String())
	}

	return
}

func (vg ValueGetter) GetBytesSlice(key string, def ...[][]byte) (res [][]byte) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case [][]byte:
		res = v
	case []interface{}:
		sl := make([][]byte, len(v))
		for i, val := range v {
			switch str := val.(type) {
			case []byte:
				sl[i] = str
			case string:
				sl[i] = []byte(str)
			case fmt.Stringer:
				sl[i] = []byte(str.String())
			default:
				return
			}
		}
		res = sl
	}

	return
}

// all ints (e.g. int8, int16, int, int32, int64) are stored as int64
func (vg ValueGetter) GetInt(key string, def ...int) int {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int(vg.GetInt64(key, i64))
}

func (vg ValueGetter) GetIntSlice(key string, def ...[]int) (res []int) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []int:
		res = v
	case []int64:
		res = make([]int, len(v))
		for i, val := range v {
			res[i] = int(val)
		}
	case []interface{}:
		sl := make([]int, len(v))
		for i, val := range v {
			if tint64, err := CastInt64(val); err == nil {
				sl[i] = int(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetInt8(key string, def ...int8) int8 {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int8(vg.GetInt64(key, i64))
}

func (vg ValueGetter) GetInt8Slice(key string, def ...[]int8) (res []int8) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []int8:
		res = v
	case []int64:
		res = make([]int8, len(v))
		for i, val := range v {
			res[i] = int8(val)
		}
	case []interface{}:
		sl := make([]int8, len(v))
		for i, val := range v {
			if tint64, err := CastInt64(val); err == nil {
				sl[i] = int8(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetInt16(key string, def ...int16) int16 {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int16(vg.GetInt64(key, i64))
}

func (vg ValueGetter) GetInt16Slice(key string, def ...[]int16) (res []int16) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []int16:
		res = v
	case []int64:
		res = make([]int16, len(v))
		for i, val := range v {
			res[i] = int16(val)
		}
	case []interface{}:
		sl := make([]int16, len(v))
		for i, val := range v {
			if tint64, err := CastInt64(val); err == nil {
				sl[i] = int16(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetInt32(key string, def ...int32) int32 {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int32(vg.GetInt64(key, i64))
}

func (vg ValueGetter) GetInt32Slice(key string, def ...[]int32) (res []int32) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []int32:
		res = v
	case []int64:
		res = make([]int32, len(v))
		for i, val := range v {
			res[i] = int32(val)
		}
	case []interface{}:
		sl := make([]int32, len(v))
		for i, val := range v {
			if tint64, err := CastInt64(val); err == nil {
				sl[i] = int32(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetInt64(key string, def ...int64) (res int64) {
	if len(def) > 0 {
		res = def[0]
	}

	if tint64, err := CastInt64(vg.Get(key)); err == nil {
		res = tint64
	}

	return
}

func (vg ValueGetter) GetInt64Slice(key string, def ...[]int64) (res []int64) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []int64:
		res = v
	case []interface{}:
		sl := make([]int64, len(v))
		for i, val := range v {
			if tint64, err := CastInt64(val); err == nil {
				sl[i] = tint64
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetUint(key string, def ...uint) uint {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint(vg.GetUint64(key, ui64))
}

func (vg ValueGetter) GetUintSlice(key string, def ...[]uint) (res []uint) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []uint:
		res = v
	case []int64:
		res = make([]uint, len(v))
		for i, val := range v {
			res[i] = uint(val)
		}
	case []uint64:
		res = make([]uint, len(v))
		for i, val := range v {
			res[i] = uint(val)
		}
	case []interface{}:
		sl := make([]uint, len(v))
		for i, val := range v {
			if tint64, err := CastUint64(val); err == nil {
				sl[i] = uint(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetUint8(key string, def ...uint8) uint8 {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint8(vg.GetUint64(key, ui64))
}

func (vg ValueGetter) GetUint8Slice(key string, def ...[]uint8) (res []uint8) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []uint8:
		res = v
	case []int64:
		res = make([]uint8, len(v))
		for i, val := range v {
			res[i] = uint8(val)
		}
	case []uint64:
		res = make([]uint8, len(v))
		for i, val := range v {
			res[i] = uint8(val)
		}
	case []interface{}:
		sl := make([]uint8, len(v))
		for i, val := range v {
			if tint64, err := CastUint64(val); err == nil {
				sl[i] = uint8(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetUint16(key string, def ...uint16) uint16 {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint16(vg.GetUint64(key, ui64))
}

func (vg ValueGetter) GetUint16Slice(key string, def ...[]uint16) (res []uint16) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []uint16:
		res = v
	case []int64:
		res = make([]uint16, len(v))
		for i, val := range v {
			res[i] = uint16(val)
		}
	case []uint64:
		res = make([]uint16, len(v))
		for i, val := range v {
			res[i] = uint16(val)
		}
	case []interface{}:
		sl := make([]uint16, len(v))
		for i, val := range v {
			if tint64, err := CastUint64(val); err == nil {
				sl[i] = uint16(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetUint32(key string, def ...uint32) uint32 {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint32(vg.GetUint64(key, ui64))
}

func (vg ValueGetter) GetUint32Slice(key string, def ...[]uint32) (res []uint32) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []uint32:
		res = v
	case []int64:
		res = make([]uint32, len(v))
		for i, val := range v {
			res[i] = uint32(val)
		}
	case []uint64:
		res = make([]uint32, len(v))
		for i, val := range v {
			res[i] = uint32(val)
		}
	case []interface{}:
		sl := make([]uint32, len(v))
		for i, val := range v {
			if tint64, err := CastUint64(val); err == nil {
				sl[i] = uint32(tint64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetUint64(key string, def ...uint64) (res uint64) {
	if len(def) > 0 {
		res = def[0]
	}

	if tuint64, err := CastUint64(vg.Get(key)); err == nil {
		res = tuint64
	}

	return
}

func (vg ValueGetter) GetUint64Slice(key string, def ...[]uint64) (res []uint64) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []uint64:
		res = v
	case []int64:
		res = make([]uint64, len(v))
		for i, val := range v {
			res[i] = uint64(val)
		}
	case []interface{}:
		sl := make([]uint64, len(v))
		for i, val := range v {
			if tint64, err := CastUint64(val); err == nil {
				sl[i] = tint64
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetFloat32(key string, def ...float32) float32 {
	var f64 float64
	if len(def) > 0 {
		f64 = float64(def[0])
	}
	return float32(vg.GetFloat64(key, f64))
}

func (vg ValueGetter) GetFloat32Slice(key string, def ...[]float32) (res []float32) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []float32:
		res = v
	case []float64:
		res = make([]float32, len(v))
		for i, val := range v {
			res[i] = float32(val)
		}
	case []interface{}:
		sl := make([]float32, len(v))
		for i, val := range v {
			if tfloat64, err := CastFloat64(val); err == nil {
				sl[i] = float32(tfloat64)
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func (vg ValueGetter) GetFloat64(key string, def ...float64) (res float64) {
	if len(def) > 0 {
		res = def[0]
	}

	if tfloat64, err := CastFloat64(vg.Get(key)); err == nil {
		res = tfloat64
	}

	return
}

func (vg ValueGetter) GetFloat64Slice(key string, def ...[]float64) (res []float64) {
	if len(def) > 0 {
		res = def[0]
	}

	switch v := vg.Get(key).(type) {
	case []float64:
		res = v
	case []interface{}:
		sl := make([]float64, len(v))
		for i, val := range v {
			if tfloat64, err := CastFloat64(val); err == nil {
				sl[i] = tfloat64
			} else {
				return
			}
		}
		res = sl
	}

	return
}

func AddDirTrailSlash(dir string) string {
	if lenp := len(dir); lenp > 0 && dir[lenp-1] != '/' && dir[lenp-1] != '\\' {
		dir = dir + "/"
	}
	return dir
}

func SaveUniqueFile(rdr io.ReadSeeker, dir string, ext string) (name string, err error) {
	var hash []byte
	if hash, err = Sha224Sum(rdr); err != nil {
		return
	}
	rdr.Seek(0, 0)

	name = fmt.Sprintf("%x%s", hash, ext)
	fname := AddDirTrailSlash(dir) + name

	if FileExists(name) {
		return
	}

	var file *os.File

	if file, err = os.Create(fname); err == nil {
		defer file.Close()
		_, err = io.Copy(file, rdr)
	}

	return
}

func Sha224Sum(rdr io.Reader) (hash []byte, err error) {
	s224 := sha256.New224()
	if _, err = io.Copy(s224, rdr); err != nil {
		return
	}
	hash = make([]byte, 0, sha256.Size224)
	hash = s224.Sum(hash)
	return
}
