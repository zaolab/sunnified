package collection

import (
	"reflect"
	"testing"
)

var (
	m1 map[string]string = map[string]string{
		"a": "a",
		"b": "b",
		"c": "c",
		"d": "d",
		"e": "e",
		"f": "f",
		"g": "g",
		"h": "h",
		"i": "i",
		"j": "j",
		"k": "k",
		"l": "l",
		"m": "m",
		"n": "n",
		"o": "o",
		"p": "p",
		"q": "q",
		"r": "r",
		"s": "s",
		"t": "t",
		"u": "u",
		"v": "v",
		"w": "w",
		"x": "x",
		"y": "y",
		"z": "z",
	}
	a1 [26]string = [...]string{
		"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	}
	s1 []string  = a1[:]
	c1 Cloneable = true
)

type Cloneable bool

func (this Cloneable) Clone() Cloneable {
	return this
}

func TestClone(t *testing.T) {
	var (
		tmpm1 interface{}
		tmpa1 [26]string
		tmps1 []string
		tmpc1 Cloneable
	)

	tmpm1 = Clone(m1)
	if !reflect.DeepEqual(m1, tmpm1) {
		t.Error("Cloning of m1 failed", tmpm1)
	}

	tmpm1 = nil
	Clone(m1, &tmpm1)
	if !reflect.DeepEqual(m1, tmpm1) {
		t.Error("Cloning of m1 via passed dest reference failed", tmpm1)
	}

	tmpa1 = Clone(a1).([26]string)
	if !reflect.DeepEqual(a1, tmpa1) {
		t.Error("Cloning of a1 failed", tmpa1)
	}

	tmpa1 = [26]string{}
	Clone(a1, &tmpa1)
	if !reflect.DeepEqual(a1, tmpa1) {
		t.Error("Cloning of a1 via passed dest reference failed", tmpa1)
	}

	tmps1 = Clone(s1).([]string)
	if !reflect.DeepEqual(s1, tmps1) {
		t.Error("Cloning of s1 failed", tmps1)
	}

	tmps1 = nil
	Clone(s1, &tmps1)
	if !reflect.DeepEqual(s1, tmps1) {
		t.Error("Cloning of s1 via passed dest reference failed", tmps1)
	}

	tmpc1 = Clone(c1).(Cloneable)
	if c1 != tmpc1 {
		t.Error("Cloning of c1 failed", tmpc1)
	}
}

func BenchmarkCloneMapMnl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tmpm1 := make(map[string]string)
		for k, v := range m1 {
			tmpm1[k] = v
		}
	}
}

func BenchmarkCloneMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Clone(m1).(map[string]string)
	}
}

func BenchmarkCloneMapRef(b *testing.B) {
	var tmpm1 map[string]string

	for i := 0; i < b.N; i++ {
		Clone(m1, &tmpm1)
	}
}

func BenchmarkCloneArrayMnl(b *testing.B) {
	var tmpa1 [26]string
	for i := 0; i < b.N; i++ {
		tmpa1 = a1
	}
	_ = tmpa1
}

func BenchmarkCloneArray(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Clone(a1).([26]string)
	}
}

func BenchmarkCloneArrayRef(b *testing.B) {
	var tmpa1 [26]string
	for i := 0; i < b.N; i++ {
		Clone(a1, &tmpa1)
	}
}

func BenchmarkCloneSliceMnl(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tmps1 := make([]string, len(s1))
		for k, v := range s1 {
			tmps1[k] = v
		}
	}
}

func BenchmarkCloneSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Clone(s1).([]string)
	}
}

func BenchmarkCloneSliceRef(b *testing.B) {
	var tmps1 []string
	for i := 0; i < b.N; i++ {
		Clone(s1, &tmps1)
	}
}
