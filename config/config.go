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

type ConfigurationSwitch []Library

func NewConfigurationFromFile(file string) (ConfigurationSwitch, error) {
	var (
		c         interface{}
		cfgswitch ConfigurationSwitch
	)

	fp, err := os.Open(file)

	if err != nil {
		return nil, err
	}

	defer fp.Close()

	jsond := json.NewDecoder(fp)
	jsond.UseNumber()
	err = jsond.Decode(&c)

	if err == nil {
		// TODO: shift namespaces key into their own map
		// "sunnified.sec" => "sunnified": map[string]interface{"sec": map[string]interface{}}
		switch conf := c.(type) {
		case []interface{}:
			cfgswitch = make(ConfigurationSwitch, len(conf))

			for i, cfg := range conf {
				m, ok := cfg.(map[string]interface{})
				if ok {
					cfgswitch[i] = Configuration(m)
				} else {
					return nil, ErrConfigFileInvalid
				}
			}
		case map[string]interface{}:
			cfgswitch = ConfigurationSwitch{Configuration(conf)}
		default:
			return nil, ErrConfigFileInvalid
		}
	} else {
		return nil, err
	}

	// func callback fsnotify
	return cfgswitch, nil
}

func (cs ConfigurationSwitch) Update() {
	// TODO: updates the configuration data whenever config file changes
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
				switch val := value.(type) {
				case int:
					value = int64(val)
				case int8:
					value = int64(val)
				case int16:
					value = int64(val)
				case int32:
					value = int64(val)
				case uint:
					value = int64(val)
				case uint8:
					value = int64(val)
				case uint16:
					value = int64(val)
				case uint32:
					value = int64(val)
				case float32:
					value = float64(val)
				}
				cfg.Set(splitname[1], value)
			}
		}
	}

	return
}

func (c Configuration) MakeBranch(name string) (cfg Library, err error) {
	i := c.Interface(name, nil)

	if i != nil {
		if val, ok := i.(map[string]interface{}); ok {
			cfg = Configuration(val)
		} else {
			err = ErrBranchKeyExists
		}
	} else {
		if strings.Contains(name, ".") {
			splitname := strings.Split(name, ".")
			cfg = c

			for i, count := 0, len(splitname); i < count; i++ {
				tmpcfg := cfg.Branch(splitname[i])

				if tmpcfg == nil {
					cfg.Set(splitname[i], make(map[string]interface{}))
					cfg = Configuration(cfg.Interface(splitname[i]).(map[string]interface{}))
				} else {
					cfg = tmpcfg
				}
			}
		}
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
		if cfg, ok := b.(map[string]interface{}); ok {
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
		if _, exists = c[key]; !exists {
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

	// TODO: refactor this; Bool(), String(), Byte()
	toInterface := make([]interface{}, len(def))

	for k, v := range def {
		toInterface[k] = v
	}

	if v, ok := c.Interface(key, toInterface...).([]interface{}); ok {
		res = v
	}

	return
}

func (c Configuration) Bool(key string, def ...bool) (res bool) {
	if len(def) > 0 {
		res = def[0]
	}

	toInterface := make([]interface{}, len(def))

	for k, v := range def {
		toInterface[k] = v
	}

	if v, ok := c.Interface(key, toInterface...).(bool); ok {
		res = v
	}

	return
}

func (c Configuration) BoolSlice(key string, def ...[]bool) (res []bool) {
	if len(def) > 0 {
		res = def[0]
	}

	if v := c.Slice(key, nil); v != nil {
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

	toInterface := make([]interface{}, len(def))

	for k, v := range def {
		toInterface[k] = v
	}

	val := c.Interface(key, toInterface...)
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

	if v := c.Slice(key, nil); v != nil {
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

	toInterface := make([]interface{}, len(def))

	for k, v := range def {
		toInterface[k] = v
	}

	if v, ok := c.Interface(key, toInterface...).(byte); ok {
		res = v
	}

	return
}

func (c Configuration) ByteSlice(key string, def ...[]byte) (res []byte) {
	if len(def) > 0 {
		res = def[0]
	}

	if v := c.Slice(key, nil); v != nil {
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
	if len(def) > 0 {
		res = def[0]
	}

	if v, ok := c.Interface(key, def).(string); ok {
		res = []byte(v)
	}

	return
}

func (c Configuration) BytesSlice(key string, def ...[][]byte) (res [][]byte) {
	if len(def) > 0 {
		res = def[0]
	}

	if v := c.Slice(key, nil); v != nil {
		sl := make([][]byte, len(v))

		for i, val := range v {
			if tstring, ok := val.(string); ok {
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]int, len(v))

		for i, val := range v {
			if tint64, err := util.CastInt64(val); err == nil {
				sl[i] = int(tint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]int8, len(v))

		for i, val := range v {
			if tint64, err := util.CastInt64(val); err == nil {
				sl[i] = int8(tint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]int16, len(v))

		for i, val := range v {
			if tint64, err := util.CastInt64(val); err == nil {
				sl[i] = int16(tint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]int32, len(v))

		for i, val := range v {
			if tint64, err := util.CastInt64(val); err == nil {
				sl[i] = int32(tint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]int64, len(v))

		var err error
		for i, val := range v {
			if sl[i], err = util.CastInt64(val); err != nil {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]uint, len(v))

		for i, val := range v {
			if tuint64, err := util.CastUint64(val); err == nil {
				sl[i] = uint(tuint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]uint8, len(v))

		for i, val := range v {
			if tuint64, err := util.CastUint64(val); err == nil {
				sl[i] = uint8(tuint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]uint16, len(v))

		for i, val := range v {
			if tuint64, err := util.CastUint64(val); err == nil {
				sl[i] = uint16(tuint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]uint32, len(v))

		for i, val := range v {
			if tuint64, err := util.CastUint64(val); err == nil {
				sl[i] = uint32(tuint64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]uint64, len(v))

		var err error
		for i, val := range v {
			if sl[i], err = util.CastUint64(val); err != nil {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]float32, len(v))

		for i, val := range v {
			if tfloat64, err := util.CastFloat64(val); err == nil {
				sl[i] = float32(tfloat64)
			} else {
				return
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

	if v := c.Slice(key, nil); v != nil {
		sl := make([]float64, len(v))

		var err error
		for i, val := range v {
			if sl[i], err = util.CastFloat64(val); err != nil {
				return
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

	if scfg, ok := valtype.FieldByName("SunnyConfig"); ok {
		cfg := c.Branch(scfg.Tag.Get("config.namespace"))

		if cfg != nil {
			val = setStructConfig(cfg, val, valtype)
		}
	}

	return val.Interface()
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

func (c Configuration) splitBranchKey(key string) (Configuration, string) {
	var keysplit []string
	var cfg = c

	if strings.Contains(key, ".") {
		keysplit = util.StringSplitLastN(key, ".", 2)

		if cfg = c.Branch(keysplit[0]).(Configuration); cfg == nil {
			return nil, ""
		}

		key = keysplit[1]
	}

	return cfg, key
}
