package util

import (
	"reflect"
	"strings"
	"testing"
)

var s1 = "This is a string. Its name is s1. It has a length x."
var sp1 = strings.Split(s1, "")
var s1tcases = [...][3]interface{}{
	[3]interface{}{"", 1, []string{s1}},
	[3]interface{}{"", -1, sp1},
	[3]interface{}{"", 100, sp1},
	[3]interface{}{"", 5, []string{"This is a string. Its name is s1. It has a lengt", "h", " ", "x", "."}},
	[3]interface{}{" ", 1, []string{s1}},
	[3]interface{}{" ", 2, []string{"This is a string. Its name is s1. It has a length", "x."}},
	[3]interface{}{" ", 4, []string{"This is a string. Its name is s1. It has", "a", "length", "x."}},
	[3]interface{}{".", 2, []string{"This is a string. Its name is s1. It has a length x", ""}},
	[3]interface{}{".", -1, strings.Split(s1, ".")},
	[3]interface{}{"It", 2, []string{"This is a string. Its name is s1. ", " has a length x."}},
	[3]interface{}{"It", 3, []string{"This is a string. ", "s name is s1. ", " has a length x."}},
	[3]interface{}{"string", 1, []string{s1}},
	[3]interface{}{"string", 2, []string{"This is a ", ". Its name is s1. It has a length x."}},
	[3]interface{}{"string", -1, strings.Split(s1, "string")},
	[3]interface{}{"This", 2, []string{"", " is a string. Its name is s1. It has a length x."}},
	[3]interface{}{"This", 5, []string{"", " is a string. Its name is s1. It has a length x."}},
	[3]interface{}{"none existance", -1, []string{s1}},
	[3]interface{}{"none existance", 5, []string{s1}},
}

func TestStringSplitLastN(t *testing.T) {
	var tmps1 []string

	for _, tcase := range s1tcases {
		tmps1 = StringSplitLastN(s1, tcase[0].(string), tcase[1].(int))
		if !reflect.DeepEqual(tmps1, tcase[2].([]string)) {
			t.Log("Sep:", tcase[0])
			t.Log("N:", tcase[1])
			t.Log("Res:", tmps1)
			t.Log("Exp:", tcase[2])
			t.Error("s1 split failed\n")
		}
	}
}
