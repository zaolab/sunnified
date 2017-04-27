package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/zaolab/sunnified/util"
	"path/filepath"
)

var ErrConfigFileInvalid = errors.New("configuration file given is invalid")
var ErrBranchKeyExists = errors.New("unable to create branch; key already exists")
var ErrValueIsBranch = errors.New("cannot set a custom value on a branch itself")

type Reader interface {
	Branch(string) Library
	ToMap() map[string]interface{}
	Exists(...string) bool
	Interface(string, ...interface{}) interface{}
	Bool(string, ...bool) bool
	Byte(string, ...byte) byte
	Bytes(string, ...[]byte) []byte
	String(string, ...string) string
	Float32(string, ...float32) float32
	Float64(string, ...float64) float64
	Int(string, ...int) int
	Int8(string, ...int8) int8
	Int16(string, ...int16) int16
	Int32(string, ...int32) int32
	Int64(string, ...int64) int64
	Uint(string, ...uint) uint
	Uint8(string, ...uint8) uint8
	Uint16(string, ...uint16) uint16
	Uint32(string, ...uint32) uint32
	Uint64(string, ...uint64) uint64
	Slice(string, ...[]interface{}) []interface{}
	BoolSlice(string, ...[]bool) []bool
	ByteSlice(string, ...[]byte) []byte
	BytesSlice(string, ...[][]byte) [][]byte
	StringSlice(string, ...[]string) []string
	Float32Slice(string, ...[]float32) []float32
	Float64Slice(string, ...[]float64) []float64
	IntSlice(string, ...[]int) []int
	Int8Slice(string, ...[]int8) []int8
	Int16Slice(string, ...[]int16) []int16
	Int32Slice(string, ...[]int32) []int32
	Int64Slice(string, ...[]int64) []int64
	UintSlice(string, ...[]uint) []uint
	Uint8Slice(string, ...[]uint8) []uint8
	Uint16Slice(string, ...[]uint16) []uint16
	Uint32Slice(string, ...[]uint32) []uint32
	Uint64Slice(string, ...[]uint64) []uint64
}

type Writer interface {
	Parse(map[string]interface{}, ...string)
	Update(Configuration)
	Set(string, interface{}) error
	MakeBranch(string) (Library, error)
}

type Library interface {
	Reader
	Writer
}

// TODO: change to struct and add in Event
// trigger event calls whenever Set/MakeBranch is called
type Configuration map[string]interface{}

func NewConfiguration() Configuration {
	return make(Configuration)
}

func NewConfigurationFromMap(m map[string]interface{}, root ...string) Configuration {
	c := NewConfiguration()
	c.Parse(m, root...)
	return c
}

func NewConfigurationFromFile(file string) (Configuration, error) {
	var (
		c   interface{}
		cfg Configuration
		root, err = os.Getwd()
	)

	if err != nil {
		return nil, err
	}

	c, err = decodeJSONFile(file, root)

	if err != nil {
		return nil, err
	}

	root = filepath.Dir(file)
	if !filepath.IsAbs(root) {
		root, err = filepath.Abs(root)
	}

	if err == nil {
		// TODO: shift namespaces key into their own map
		// "sunnified.sec" => "sunnified": map[string]interface{"sec": map[string]interface{}}
		switch conf := c.(type) {
		case []interface{}:
			cfg = NewConfiguration()

			for i := range conf {
				m, ok := conf[i].(map[string]interface{})
				if ok {
					cfg.Update(NewConfigurationFromMap(m, root))
				} else {
					return nil, ErrConfigFileInvalid
				}
			}
		case map[string]interface{}:
			cfg = NewConfigurationFromMap(conf, root)
		default:
			return nil, ErrConfigFileInvalid
		}
	} else {
		return nil, err
	}

	// func callback fsnotify
	return cfg, nil
}

func (c Configuration) Parse(m map[string]interface{}, root ...string) {
	c.parseInclude(m, root...)
	c.parseSwitch(nil)
}

func (c Configuration) Update(cc Configuration) {
	if cc == nil || reflect.ValueOf(c).Pointer() == reflect.ValueOf(cc).Pointer() {
		return
	}

	for k, v := range cc {
		if m, ok := v.(Configuration); ok {
			b := c.Branch(k)
			if b == nil {
				if newb, err := c.MakeBranch(k); err != nil {
					c.Set(k, v)
				} else {
					newb.Update(Configuration(m))
				}
			} else {
				b.Update(Configuration(m))
			}
		} else if m, ok := v.(map[string]interface{}); ok {
			b := c.Branch(k)
			if b == nil {
				if newb, err := c.MakeBranch(k); err != nil {
					c.Set(k, v)
				} else {
					newb.Update(Configuration(m))
				}
			} else {
				b.Update(Configuration(m))
			}
		} else {
			c.Set(k, v)
		}
	}
}

func (c Configuration) Set(name string, value interface{}) (err error) {
	if strings.Contains(name, ".") {
		splitname := util.StringSplitLastN(name, ".", 2)
		var cfg Library
		if cfg, err = c.MakeBranch(splitname[0]); err == nil {
			preval := cfg.Interface(splitname[1], nil)
			if preval != nil {
				if _, ismap := preval.(map[string]interface{}); ismap {
					err = ErrValueIsBranch
				}
			}

			if err == nil {
				cfg.Set(splitname[1], toBigType(value))
			}
		}
	} else {
		c[name] = toBigType(value)
	}

	return
}

func (c Configuration) MakeBranch(name string) (cfg Library, err error) {
	i := c.Interface(name, nil)

	if i != nil {
		if val, ok := i.(map[string]interface{}); ok {
			cfg = Configuration(val)
		} else if cfg, ok = i.(Configuration); ok {
			return
		} else {
			err = ErrBranchKeyExists
		}
	} else if strings.Contains(name, ".") {
		splitname := strings.Split(name, ".")
		cfg = c

		for i, count := 0, len(splitname); i < count; i++ {
			tmpcfg := cfg.Branch(splitname[i])

			if tmpcfg == nil {
				tmpcfg = NewConfiguration()
				cfg.Set(splitname[i], tmpcfg)
				cfg = tmpcfg
			} else {
				cfg = tmpcfg
			}
		}
	} else {
		cfg = NewConfiguration()
		c.Set(name, cfg)
	}

	return
}

func (c Configuration) Branch(name string) Library {
	var namesplit []string

	if strings.Contains(name, ".") {
		namesplit = strings.SplitN(name, ".", 2)
		name = namesplit[0]
	} else if name == "" {
		// do not remove, LoadConfigStruct relies on this
		return c
	}

	if b, exists := c[name]; exists {
		if cc, ok := b.(Configuration); ok {
			if namesplit != nil {
				return cc.Branch(namesplit[1])
			}

			return cc
		} else if cfg, ok := b.(map[string]interface{}); ok {
			if namesplit != nil {
				return Configuration(cfg).Branch(namesplit[1])
			}

			return Configuration(cfg)
		}
	}

	return nil
}

func (c Configuration) ToMap() map[string]interface{} {
	return map[string]interface{}(c)
}

func (c Configuration) Exists(keys ...string) (exists bool) {
	exists = true

	for _, key := range keys {
		cfg, k := c.splitBranchKey(key)
		if _, exists = cfg[k]; !exists {
			break
		}
	}

	return
}

func (c Configuration) Interface(key string, def ...interface{}) (res interface{}) {
	if len(def) > 0 {
		res = def[0]
	}

	cfg, k := c.splitBranchKey(key)

	if cfg == nil {
		return
	}

	if val, ok := cfg[k]; ok {
		res = val
	}

	return
}

func (c Configuration) Slice(key string, def ...[]interface{}) (res []interface{}) {
	if len(def) > 0 {
		res = def[0]
	}

	if v, ok := c.Interface(key).([]interface{}); ok {
		res = v
	}

	return
}

func (c Configuration) Bool(key string, def ...bool) (res bool) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.(bool); ok {
		res = v
	} else if v, ok := value.(string); ok && len(v) > 0 {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "t", "yes", "y", "1":
			res = true
		case "false", "f", "no", "n", "0":
			res = false
		}
	} else if v := c.Int64(key, -1); v == 1 || v == 0 {
		res = v == 1
	}

	return
}

func (c Configuration) BoolSlice(key string, def ...[]bool) (res []bool) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]bool); ok {
		res = v
	} else if v, ok := value.([]interface{}); ok {
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

func (c Configuration) String(key string, def ...string) (res string) {
	if len(def) > 0 {
		res = def[0]
	}

	val := c.Interface(key)
	if v, ok := val.(string); ok {
		res = v
	} else if v, ok := val.(fmt.Stringer); ok {
		res = v.String()
	}

	return
}

func (c Configuration) StringSlice(key string, def ...[]string) (res []string) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]string); ok {
		res = v
	} else if v, ok := value.([]interface{}); ok {
		sl := make([]string, len(v))

		var ok bool
		for i, val := range v {
			if sl[i], ok = val.(string); !ok {
				if stringer, ok := val.(fmt.Stringer); ok {
					sl[i] = stringer.String()
				} else {
					return
				}
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Byte(key string, def ...byte) (res byte) {
	if len(def) > 0 {
		res = def[0]
	}

	if v, ok := c.Interface(key).(byte); ok {
		res = v
	}

	return
}

func (c Configuration) ByteSlice(key string, def ...[]byte) (res []byte) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]byte); ok {
		res = v
	} else if v, ok := value.([]interface{}); ok {
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

func (c Configuration) Bytes(key string, def ...[]byte) (res []byte) {
	if v, ok := c.Interface(key, def).(string); ok {
		res = []byte(v)
	} else {
		res = c.ByteSlice(key, def...)
	}

	return
}

func (c Configuration) BytesSlice(key string, def ...[][]byte) (res [][]byte) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([][]byte); ok {
		res = v
	} else if v, ok := value.([]interface{}); ok {
		sl := make([][]byte, len(v))

		for i, val := range v {
			if tbytes, ok := val.([]byte); ok {
				sl[i] = tbytes
			} else if tstring, ok := val.(string); ok {
				sl[i] = []byte(tstring)
			} else {
				return
			}
		}

		res = sl
	}

	return
}

// all ints (e.g. int8, int16, int, int32, int64) are stored as int64
func (c Configuration) Int(key string, def ...int) int {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int(c.Int64(key, i64))
}

func (c Configuration) IntSlice(key string, def ...[]int) (res []int) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]int); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]int, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = int(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = int(iv.Int())
			} else {
				sl[i] = int(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Int8(key string, def ...int8) int8 {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int8(c.Int64(key, i64))
}

func (c Configuration) Int8Slice(key string, def ...[]int8) (res []int8) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]int8); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]int8, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = int8(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = int8(iv.Int())
			} else {
				sl[i] = int8(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Int16(key string, def ...int16) int16 {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int16(c.Int64(key, i64))
}

func (c Configuration) Int16Slice(key string, def ...[]int16) (res []int16) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]int16); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]int16, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = int16(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = int16(iv.Int())
			} else {
				sl[i] = int16(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Int32(key string, def ...int32) int32 {
	var i64 int64
	if len(def) > 0 {
		i64 = int64(def[0])
	}
	return int32(c.Int64(key, i64))
}

func (c Configuration) Int32Slice(key string, def ...[]int32) (res []int32) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]int32); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]int32, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = int32(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = int32(iv.Int())
			} else {
				sl[i] = int32(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Int64(key string, def ...int64) (res int64) {
	if len(def) > 0 {
		res = def[0]
	}

	if tint64, err := util.CastInt64(c.Interface(key, def)); err == nil {
		res = tint64
	}

	return
}

func (c Configuration) Int64Slice(key string, def ...[]int64) (res []int64) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]int64); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]int64, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = int64(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = int64(iv.Int())
			} else {
				sl[i] = int64(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Uint(key string, def ...uint) uint {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint(c.Uint64(key, ui64))
}

func (c Configuration) UintSlice(key string, def ...[]uint) (res []uint) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]uint); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]uint, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = uint(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = uint(iv.Int())
			} else {
				sl[i] = uint(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Uint8(key string, def ...uint8) uint8 {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint8(c.Uint64(key, ui64))
}

func (c Configuration) Uint8Slice(key string, def ...[]uint8) (res []uint8) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]uint8); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]uint8, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = uint8(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = uint8(iv.Int())
			} else {
				sl[i] = uint8(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Uint16(key string, def ...uint16) uint16 {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint16(c.Uint64(key, ui64))
}

func (c Configuration) Uint16Slice(key string, def ...[]uint16) (res []uint16) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]uint16); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]uint16, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = uint16(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = uint16(iv.Int())
			} else {
				sl[i] = uint16(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Uint32(key string, def ...uint32) uint32 {
	var ui64 uint64
	if len(def) > 0 {
		ui64 = uint64(def[0])
	}
	return uint32(c.Uint64(key, ui64))
}

func (c Configuration) Uint32Slice(key string, def ...[]uint32) (res []uint32) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]uint32); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]uint32, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = uint32(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = uint32(iv.Int())
			} else {
				sl[i] = uint32(iv.Uint())
			}
		}

		res = sl
	}

	return
}

// all unsigned int64 are also stored int64... unless overflowed...
func (c Configuration) Uint64(key string, def ...uint64) (res uint64) {
	if len(def) > 0 {
		res = def[0]
	}

	if tuint64, err := util.CastUint64(c.Interface(key, def)); err == nil {
		res = tuint64
	}

	return
}

func (c Configuration) Uint64Slice(key string, def ...[]uint64) (res []uint64) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]uint64); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]uint64, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = uint64(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = uint64(iv.Int())
			} else {
				sl[i] = uint64(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Float32(key string, def ...float32) float32 {
	var f64 float64
	if len(def) > 0 {
		f64 = float64(def[0])
	}
	return float32(c.Float64(key, f64))
}

func (c Configuration) Float32Slice(key string, def ...[]float32) (res []float32) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]float32); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]float32, vv.Len())

		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = float32(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = float32(iv.Int())
			} else {
				sl[i] = float32(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) Float64(key string, def ...float64) (res float64) {
	if len(def) > 0 {
		res = def[0]
	}

	if tfloat64, err := util.CastFloat64(c.Interface(key, def)); err == nil {
		res = tfloat64
	}

	return
}

func (c Configuration) Float64Slice(key string, def ...[]float64) (res []float64) {
	if len(def) > 0 {
		res = def[0]
	}

	var value = c.Interface(key)

	if v, ok := value.([]float64); ok {
		res = v
	} else if vv := reflect.ValueOf(value); vv.Kind() == reflect.Slice || vv.Kind() == reflect.Array {
		sl := make([]float64, vv.Len())
		defer func() { recover() }()

		for i := range sl {
			iv := vv.Index(i)
			if k := iv.Kind(); k == reflect.Float32 || k == reflect.Float64 {
				sl[i] = float64(iv.Float())
			} else if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 ||
				k == reflect.Int32 || k == reflect.Int64 {

				sl[i] = float64(iv.Int())
			} else {
				sl[i] = float64(iv.Uint())
			}
		}

		res = sl
	}

	return
}

func (c Configuration) LoadStruct(namespace string, st interface{}) interface{} {
	cfg := c.Branch(namespace)

	if cfg != nil {
		val := reflect.ValueOf(st)
		return setStructConfig(cfg, val, val.Type()).Interface()
	}

	return st
}

func (c Configuration) LoadConfigStruct(st interface{}) interface{} {
	val := reflect.ValueOf(st)
	valtype := val.Type()

	if valtype.Kind() == reflect.Ptr {
		valtype = val.Elem().Type()
	}

	if scfg, ok := valtype.FieldByName("SunnyConfig"); ok {
		cfg := c.Branch(scfg.Tag.Get("config.namespace"))

		if cfg != nil {
			val = setStructConfig(cfg, val, val.Type())
		}
	}

	return val.Interface()
}

func (c Configuration) parseInclude(m map[string]interface{}, root ...string) {
	for k, v := range m {
		if mval, ok := v.(map[string]interface{}); ok && k != "__switch__" {
			cfg := NewConfiguration()
			cfg.parseInclude(mval, root...)
			c.Set(k, cfg)
		} else if k != "__include__" {
			c.Set(k, v)
		}
	}

	var rootpath string
	if len(root) > 0 {
		rootpath = root[0]
		if !filepath.IsAbs(rootpath) {
			if r, err := filepath.Abs(rootpath); err == nil {
				rootpath = r
			}
		}
	} else {
		rootpath, _ = os.Getwd()
	}

	if inc, ok := m["__include__"]; ok {
		switch v := inc.(type) {
		case string:
			js, err := decodeJSONFile(v, rootpath)
			if err != nil {
				break
			}
			c.Update(parseJSONToConfiguration(js,
				filepath.Dir(filepath.Join(rootpath, v))))
		case []interface{}:
			for i := range v {
				if fname, ok := v[i].(string); ok {
					js, err := decodeJSONFile(fname, rootpath)
					if err != nil {
						continue
					}
					c.Update(parseJSONToConfiguration(js,
						filepath.Dir(filepath.Join(rootpath, fname))))
				}
			}
		case []string:
			for i := range v {
				js, err := decodeJSONFile(v[i], rootpath)
				if err != nil {
					continue
				}
				c.Update(parseJSONToConfiguration(js,
					filepath.Dir(filepath.Join(rootpath, v[i]))))
			}
		}
	}
}

func (c Configuration) parseSwitch(cc Configuration) {
	if cc == nil {
		cc = c
	}

	if v, ok := cc["__switch__"]; ok {
		if m2, ok := v.(map[string]interface{}); ok {
			v = Configuration(m2)
		}
		if m2, ok := v.(Configuration); !ok {
			goto switchcleanup
		} else if key, ok := m2["__key__"]; !ok {
			goto switchcleanup
		} else if s, ok := key.(string); ok {
			selected := c.String(s, "")
			if selected == "" {
				if def, ok := m2["__default__"]; ok {
					selected, _ = def.(string)
				}
			}

			if caseVal, ok := m2[selected]; ok {
				if cfg, ok := caseVal.(Configuration); ok {
					cc.Update(cfg)
				} else if cfg, ok := caseVal.(map[string]interface{}); ok {
					cc.Update(Configuration(cfg))
				}
			}
		}

	switchcleanup:
		delete(cc, "__switch__")
	}

	for _, v := range cc {
		if cfg, ok := v.(Configuration); ok {
			c.parseSwitch(cfg)
		}
	}
}

func (c Configuration) splitBranchKey(key string) (Configuration, string) {
	var keysplit []string
	var ok bool
	var cfg = c

	if strings.Contains(key, ".") {
		keysplit = util.StringSplitLastN(key, ".", 2)

		if cfg, ok = c.Branch(keysplit[0]).(Configuration); !ok || cfg == nil {
			return nil, ""
		}

		key = keysplit[1]
	}

	return cfg, key
}

func decodeJSONFile(fname string, root string) (interface{}, error) {
	if !filepath.IsAbs(fname) {
		fname = filepath.Join(root, fname)
	}

	var c interface{}
	fp, err := os.Open(fname)

	if err != nil {
		return nil, err
	}

	defer fp.Close()

	jsond := json.NewDecoder(fp)
	err = jsond.Decode(&c)

	return c, nil
}

func parseJSONToConfiguration(js interface{}, root ...string) (cfg Configuration) {
	switch conf := js.(type) {
	case []interface{}:
		cfg = NewConfiguration()

		for i := range conf {
			if m, ok := conf[i].(map[string]interface{}); ok {
				cc := NewConfiguration()
				cc.parseInclude(m, root...)
				cfg.Update(cc)
			}
		}
	case map[string]interface{}:
		cfg = NewConfiguration()
		cfg.parseInclude(conf, root...)
	}

	return
}

func setStructConfig(cfg Library, val reflect.Value, valtype reflect.Type) reflect.Value {
	var isPtr = false
	var retval reflect.Value

	if valtype.Kind() == reflect.Ptr {
		retval = val
		isPtr = true
		val = val.Elem()
		valtype = val.Type()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		if !field.CanSet() {
			continue
		}

		fieldtype := valtype.Field(i)
		name := strings.ToLower(fieldtype.Name)

		switch field.Type().Kind() {
		case reflect.String:
			field.SetString(cfg.String(name, fieldtype.Tag.Get("config.default")))
		case reflect.Bool:
			field.SetBool(cfg.Bool(name, strings.ToLower(strings.TrimSpace(fieldtype.Tag.Get("config.default"))) == "true"))
		case reflect.Slice:
			switch field.Type().Elem().Kind() {
			case reflect.Uint8:
				field.SetBytes(cfg.Bytes(name, []byte(fieldtype.Tag.Get("config.default"))))
			default:
			}
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			i, _ := strconv.ParseInt(fieldtype.Tag.Get("config.default"), 10, 64)
			field.SetInt(cfg.Int64(name, i))
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			i, _ := strconv.ParseUint(fieldtype.Tag.Get("config.default"), 10, 64)
			field.SetUint(cfg.Uint64(name, i))
		case reflect.Float32, reflect.Float64:
			i, _ := strconv.ParseFloat(fieldtype.Tag.Get("config.default"), 64)
			field.SetFloat(cfg.Float64(name, i))
		default:
		}
	}

	if isPtr {
		return retval
	}

	return val
}

func toBigType(v interface{}) interface{} {
	switch val := v.(type) {
	case int:
		v = int64(val)
	case int8:
		v = int64(val)
	case int16:
		v = int64(val)
	case int32:
		v = int64(val)
	case uint:
		v = uint64(val)
	case uint16:
		v = uint64(val)
	case uint32:
		v = uint64(val)
	case float32:
		v = float64(val)
	}
	return v
}
