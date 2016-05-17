package collection

import (
	"reflect"

	"github.com/zaolab/sunnified/util"
)

// usage
// for iter := li.Iterator(); iter.Next(&i, &value); {
//     ...
//     i, val := iter.GetI() // alternatively use this instead of .Next(&i, &value)
// }
type Iterator interface {
	Get() (interface{}, interface{})
	Next(...interface{}) bool
	Reset()
}

type NumIterator interface {
	Iterator
	GetI() (int, interface{})
}

type PopIterator interface {
	NumIterator
	PopNext(...interface{}) bool
}

type Array interface {
	Append(value interface{}) Array
	Contains(value ...interface{}) bool
	Extend(values []interface{})
	ExtendArray(Array)
	First() interface{}
	Last() interface{}
	Index(value interface{}) int
	Indexes(value interface{}) []int
	LastIndex(value interface{}) int
	Set(index int, value interface{}) Array
	Insert(index int, value interface{}) Array
	MapValue(index int, value interface{}) interface{}
	Len() int
	Pop() interface{}
	Remove(value interface{})
	RemoveAt(index int) interface{}
	Reverse()
	Iterator() NumIterator
	String() string
	ToSlice() []interface{}
	Clear()
}

type ExtendedArray interface {
	Array
	Less(x int, y int) bool
	Swap(x int, y int)
	Sort(less func(x interface{}, y interface{}) bool)
	Filter(func(interface{}) bool) ExtendedArray
	Foreach(func(int, interface{}) bool)
	IsMatch(func(interface{}) bool) bool
	Match(func(interface{}) bool) (index int, value interface{})
	Map(func(interface{}) interface{}) ExtendedArray
	Reduce(f func(interface{}, interface{}) interface{}, init interface{}) interface{}
	Transaction(func(ExtendedArray) bool)
}

type Dictionary interface {
	Contains(values ...interface{}) bool
	HasKey(key interface{}) bool
	KeyOf(value interface{}) interface{}
	Keys() []interface{}
	Values() []interface{}
	KeysValues() (keys []interface{}, values []interface{})
	Pairs() [][2]interface{}
	Pop() (key interface{}, value interface{})
	Len() int
	MapValue(key interface{}, value interface{}) interface{}
	Remove(value interface{}) []interface{}
	RemoveAt(key interface{}) interface{}
	Set(key interface{}, value interface{}) Dictionary
	SetDefault(key interface{}, value interface{}) Dictionary
	ToMap() map[interface{}]interface{}
	Update(map[interface{}]interface{})
	UpdateDictionary(Dictionary)
	Iterator() Iterator
	String() string
	Clear()
}

type ExtendedDictionary interface {
	Dictionary
	Foreach(f func(interface{}, interface{}) bool)
	IsMatch(func(interface{}) bool) bool
	Match(f func(interface{}) bool) (key interface{}, value interface{})
	Transaction(f func(ExtendedDictionary) bool)
}

func Clone(c interface{}, cloned ...interface{}) (res interface{}) {
	cVal := reflect.ValueOf(c)

	switch cVal.Kind() {
	case reflect.Array:
		// arrays are passed by value...
		if len(cloned) > 0 {
			util.MapValues(cloned, c)
		}
		return c
	case reflect.Slice:
		dst := reflect.MakeSlice(cVal.Type(), cVal.Len(), cVal.Len())
		reflect.Copy(dst, cVal)
		res = dst.Interface()
	case reflect.Map:
		dst := reflect.MakeMap(cVal.Type())
		keys := cVal.MapKeys()
		for _, key := range keys {
			dst.SetMapIndex(key, cVal.MapIndex(key))
		}
		res = dst.Interface()
	default:
		mVal := cVal.MethodByName("Clone")
		if mVal.IsValid() && mVal.Type().NumIn() == 0 && mVal.Type().NumOut() == 1 {
			res = mVal.Call(nil)[0].Interface()
		}
	}

	if res != nil && len(cloned) > 0 {
		util.MapValues(cloned, res)
	}

	return res
}

func Map(slice interface{}, f func(interface{}) interface{}, dest ...interface{}) (res interface{}) {
	var (
		sliceval = reflect.ValueOf(slice)
		lenslice = sliceval.Len()
	)

	if len(dest) > 0 && dest[0] != nil {
		newslice := fixSliceLen(reflect.Indirect(reflect.ValueOf(dest[0])), lenslice)

		for i := 0; i < lenslice; i++ {
			newslice.Index(i).Set(reflect.ValueOf(f(sliceval.Index(i).Interface())).Elem())
		}

		res = newslice.Interface()

		if len(dest) > 1 {
			util.MapValues(dest[1:], res)
		}
	} else {
		newslice := make([]interface{}, lenslice)

		for i := 0; i < lenslice; i++ {
			newslice[i] = f(sliceval.Index(i).Interface())
		}

		res = newslice
	}

	return
}

func fixSliceLen(slice reflect.Value, lenslice int) reflect.Value {
	if slice.Len() != lenslice {
		if slice.Cap() >= lenslice {
			slice.SetLen(lenslice)
		} else {
			slice.Set(reflect.MakeSlice(slice.Type(), lenslice, lenslice))
		}
	}
	return slice
}

func Filter(slice interface{}, f func(interface{}) bool, dest ...interface{}) (res interface{}) {
	var (
		sliceval = reflect.ValueOf(slice)
		lenslice = sliceval.Len()
		newslice = reflect.MakeSlice(sliceval.Type(), lenslice, lenslice)
		count    = 0
	)

	for i := 0; i < lenslice; i++ {
		val := sliceval.Index(i)
		if f(val.Interface()) {
			newslice.Index(count).Set(val)
			count++
		}
	}

	newslice.SetLen(count)
	res = newslice.Interface()
	util.MapValues(dest, res)

	return
}

func Reduce(slice interface{}, f func(interface{}, interface{}) interface{}, init interface{}, dest ...interface{}) (res interface{}) {
	var (
		sliceval = reflect.ValueOf(slice)
		lenslice = sliceval.Len()
		i        = 0
	)

	if init != nil {
		res = init
	} else if lenslice > 0 {
		res = sliceval.Index(0)
		i = 1
	} else {
		return
	}

	for ; i < lenslice; i++ {
		res = f(res, sliceval.Index(i).Interface())
	}

	util.MapValues(dest, res)
	return
}

func IsMatch(slice interface{}, f func(interface{}) bool) bool {
	var sliceval = reflect.ValueOf(slice)

	switch sliceval.Kind() {
	case reflect.Array, reflect.Slice:
		for i, lenslice := 0, sliceval.Len(); i < lenslice; i++ {
			if f(sliceval.Index(i).Interface()) {
				return true
			}
		}
	case reflect.Map:
		keys := sliceval.MapKeys()
		for _, key := range keys {
			if f(sliceval.MapIndex(key).Interface()) {
				return true
			}
		}
	}

	return false
}

func Match(slice interface{}, f func(interface{}) bool) (interface{}, interface{}) {
	var sliceval = reflect.ValueOf(slice)

	switch sliceval.Kind() {
	case reflect.Array, reflect.Slice:
		for i, lenslice := 0, sliceval.Len(); i < lenslice; i++ {
			val := sliceval.Index(i).Interface()
			if f(val) {
				return i, val
			}
		}
	case reflect.Map:
		keys := sliceval.MapKeys()
		for _, key := range keys {
			val := sliceval.MapIndex(key).Interface()
			if f(val) {
				return key.Interface(), val
			}
		}
	}

	return nil, nil
}

func Foreach(slice interface{}, f func(interface{}, interface{}) bool) {
	var sliceval = reflect.ValueOf(slice)

	switch sliceval.Kind() {
	case reflect.Array, reflect.Slice:
		for i, lenslice := 0, sliceval.Len(); i < lenslice; i++ {
			if !f(i, sliceval.Index(i).Interface()) {
				break
			}
		}
	case reflect.Map:
		keys := sliceval.MapKeys()
		for _, key := range keys {
			if !f(key.Interface(), sliceval.MapIndex(key).Interface()) {
				break
			}
		}
	}
}
