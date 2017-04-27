package config

import (
	"reflect"
	"testing"
)

func TestConfig(t *testing.T) {
	var (
		skey    = "string.value"
		ikey    = "int.value"
		uikey   = "uint.value"
		i64key  = "int64.value"
		ui64key = "uint64.value"
		bkey    = "bool.value"
		btkey   = "byte.value"
		fkey    = "float.value"
		f64key  = "float64.value"
		btskey  = "byte.slice.value"
		slkey   = "slice.value"
		nonkey  = "~!@#$%"
		branch  = "branchname.nest"

		sval            = "12345678"
		ival            = 12345678
		uival   uint32  = 4294967295
		i64val  int64   = -9223372036854775808
		ui64val uint64  = 18446744073709551615
		bval            = true
		btval           = byte(82)
		fval    float32 = 123.456
		f64val          = 123.23787283123123
		btsval          = []byte{1, 2, 3, 4, 5, 6, 7, 8}
		slval           = []interface{}{"12345678", 12345678}
	)

	c := NewConfiguration()
	c.Set(skey, sval)
	c.Set(ikey, ival)
	c.Set(uikey, uival)
	c.Set(i64key, i64val)
	c.Set(ui64key, ui64val)
	c.Set(bkey, bval)
	c.Set(btkey, btval)
	c.Set(fkey, fval)
	c.Set(f64key, f64val)
	c.Set(btskey, btsval)
	c.Set(slkey, slval)
	b, err := c.MakeBranch(branch)

	if !c.Exists(skey, ikey, uikey, i64key, ui64key, bkey, btkey, fkey, f64key, btskey, slkey) {
		t.Error("Keys do not exists")
	} else if c.Exists(nonkey) {
		t.Error("Key exists")
	}

	if c.String(skey) != sval {
		t.Log("Key:", skey)
		t.Log("Expected:", sval)
		t.Log("Result:", c.String(skey))
		t.Error("Expected value and result does not match")
	} else if c.String(nonkey, "not exists") != "not exists" {
		t.Log("Key:", nonkey)
		t.Log("Expected:", "not exists")
		t.Log("Result:", c.String(nonkey, "not exists"))
		t.Error("Expected value and result does not match")
	}
	if !reflect.DeepEqual(c.Bytes(skey), []byte(sval)) {
		t.Log("Key:", skey)
		t.Log("Expected:", []byte(sval))
		t.Log("Result:", c.Bytes(skey))
		t.Error("Expected value and result does not match")
	} else if !reflect.DeepEqual(c.Bytes(nonkey, []byte("not exists b")), []byte("not exists b")) {
		t.Log("Key:", nonkey)
		t.Log("Expected:", []byte("not exists b"))
		t.Log("Result:", c.Bytes(nonkey, []byte("not exists b")))
		t.Error("Expected value and result does not match")
	}

	if c.Int(ikey) != ival {
		t.Log("Key:", ikey)
		t.Log("Expected:", ival)
		t.Log("Result:", c.Int(ikey))
		t.Error("Expected value and result does not match")
	} else if c.Int(nonkey, 8282) != 8282 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 8282)
		t.Log("Result:", c.Int(nonkey, 8282))
		t.Error("Expected value and result does not match")
	}
	if c.Int32(ikey) != int32(ival) {
		t.Log("Key:", ikey)
		t.Log("Expected:", int32(ival))
		t.Log("Result:", c.Int32(ikey))
		t.Error("Expected value and result does not match")
	} else if c.Int32(nonkey, 828232) != 828232 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 828232)
		t.Log("Result:", c.Int32(nonkey, 828232))
		t.Error("Expected value and result does not match")
	}

	if c.Uint32(uikey) != uival {
		t.Log("Key:", uikey)
		t.Log("Expected:", uival)
		t.Log("Result:", c.Uint32(uikey))
		t.Error("Expected value and result does not match")
	} else if c.Uint32(nonkey, 8282132) != 8282132 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 8282132)
		t.Log("Result:", c.Uint32(nonkey, 8282132))
		t.Error("Expected value and result does not match")
	}
	if c.Uint(uikey) != uint(uival) {
		t.Log("Key:", uikey)
		t.Log("Expected:", uint(uival))
		t.Log("Result:", c.Uint(uikey))
		t.Error("Expected value and result does not match")
	} else if c.Uint(nonkey, 82821) != 82821 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 82821)
		t.Log("Result:", c.Uint(nonkey, 82821))
		t.Error("Expected value and result does not match")
	}

	if c.Int64(i64key) != i64val {
		t.Log("Key:", i64key)
		t.Log("Expected:", i64val)
		t.Log("Result:", c.Int64(i64key))
		t.Error("Expected value and result does not match")
	} else if c.Int64(nonkey, 828264) != 828264 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 828264)
		t.Log("Result:", c.Int64(nonkey, 828264))
		t.Error("Expected value and result does not match")
	}

	if c.Uint64(ui64key) != ui64val {
		t.Log("Key:", ui64key)
		t.Log("Expected:", ui64val)
		t.Log("Result:", c.Uint64(ui64key))
		t.Error("Expected value and result does not match")
	} else if c.Uint64(nonkey, 8282164) != 8282164 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 8282164)
		t.Log("Result:", c.Uint64(nonkey, 8282164))
		t.Error("Expected value and result does not match")
	}

	if c.Bool(bkey) != bval {
		t.Log("Key:", bkey)
		t.Log("Expected:", bval)
		t.Log("Result:", c.Bool(bkey))
		t.Error("Expected value and result does not match")
	} else if c.Bool(nonkey, true) != true {
		t.Log("Key:", nonkey)
		t.Log("Expected:", true)
		t.Log("Result:", c.Bool(nonkey, true))
		t.Error("Expected value and result does not match")
	}

	if c.Byte(btkey) != btval {
		t.Log("Key:", btkey)
		t.Log("Expected:", btval)
		t.Log("Result:", c.Byte(btkey))
		t.Error("Expected value and result does not match")
	} else if c.Byte(nonkey, 8) != 8 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 8)
		t.Log("Result:", c.Byte(nonkey, 8))
		t.Error("Expected value and result does not match")
	}
	if c.Int8(btkey) != int8(btval) {
		t.Log("Key:", btkey)
		t.Log("Expected:", int8(btval))
		t.Log("Result:", c.Int8(btkey))
		t.Error("Expected value and result does not match")
	} else if c.Int8(nonkey, 9) != 9 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 9)
		t.Log("Result:", c.Int8(nonkey, 9))
		t.Error("Expected value and result does not match")
	}
	if c.Int16(btkey) != int16(btval) {
		t.Log("Key:", btkey)
		t.Log("Expected:", int16(btval))
		t.Log("Result:", c.Int16(btkey))
		t.Error("Expected value and result does not match")
	} else if c.Int16(nonkey, 8216) != 8216 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 8216)
		t.Log("Result:", c.Int16(nonkey, 8216))
		t.Error("Expected value and result does not match")
	}
	if c.Uint8(btkey) != btval {
		t.Log("Key:", btkey)
		t.Log("Expected:", btval)
		t.Log("Result:", c.Uint8(btkey))
		t.Error("Expected value and result does not match")
	} else if c.Uint8(nonkey, 81) != 81 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 81)
		t.Log("Result:", c.Uint8(nonkey, 81))
		t.Error("Expected value and result does not match")
	}
	if c.Uint16(btkey) != uint16(btval) {
		t.Log("Key:", btkey)
		t.Log("Expected:", uint16(btval))
		t.Log("Result:", c.Uint16(btkey))
		t.Error("Expected value and result does not match")
	} else if c.Uint16(nonkey, 8216) != 8216 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 8216)
		t.Log("Result:", c.Uint16(nonkey, 8216))
		t.Error("Expected value and result does not match")
	}

	if c.Float32(fkey) != fval {
		t.Log("Key:", fkey)
		t.Log("Expected:", fval)
		t.Log("Result:", c.Float32(fkey))
		t.Error("Expected value and result does not match")
	} else if c.Float32(nonkey, 82.32) != 82.32 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 82.32)
		t.Log("Result:", c.Float32(nonkey, 82.32))
		t.Error("Expected value and result does not match")
	}
	if c.Float64(f64key) != f64val {
		t.Log("Key:", f64key)
		t.Log("Expected:", f64val)
		t.Log("Result:", c.Float64(f64key))
		t.Error("Expected value and result does not match")
	} else if c.Float64(nonkey, 82.64) != 82.64 {
		t.Log("Key:", nonkey)
		t.Log("Expected:", 82.64)
		t.Log("Result:", c.Float64(nonkey, 82.64))
		t.Error("Expected value and result does not match")
	}

	if !reflect.DeepEqual(c.ByteSlice(btskey), btsval) {
		t.Log("Key:", btskey)
		t.Log("Expected:", btsval)
		t.Log("Result:", c.Byte(btskey))
		t.Error("Expected value and result does not match")
	} else if !reflect.DeepEqual(c.ByteSlice(nonkey, []byte("not exists bs")), []byte("not exists bs")) {
		t.Log("Key:", nonkey)
		t.Log("Expected:", []byte("not exists bs"))
		t.Log("Result:", c.ByteSlice(nonkey, []byte("not exists bs")))
		t.Error("Expected value and result does not match")
	}

	if err != nil {
		t.Error("MakeBranch error:", err)
	} else {
		b.Set(skey, sval)

		if b.String(skey) != sval {
			t.Log("Key:", branch, skey)
			t.Log("Expected:", sval)
			t.Log("Result:", b.String(skey))
			t.Error("Expected value and result does not match")
		} else if c.String(branch+"."+skey) != sval {
			t.Log("Key:", branch, skey)
			t.Log("Expected:", sval)
			t.Log("Result:", b.String(skey))
			t.Error("Expected value and result does not match")
		} else if c.Branch(branch).String(skey) != sval {
			t.Log("Key:", branch, skey)
			t.Log("Expected:", sval)
			t.Log("Result:", b.String(skey))
			t.Error("Expected value and result does not match")
		}
	}
}

func TestConfigMap(t *testing.T) {
	m := map[string]interface{}{
		"name":    "sunnified",
		"version": "b",
		"branch": map[string]interface{}{
			"key":   "value",
			"int":   42,
			"float": 32.12,
			"bool":  true,
		},
		"switches": map[string]interface{}{
			"__switch__": map[string]interface{}{
				"__key__":     "version",
				"__default__": "c",
				"a": map[string]interface{}{
					"1": "1",
					"2": "2",
				},
				"b": map[string]interface{}{
					"1": "one",
					"2": "two",
				},
				"c": map[string]interface{}{
					"1": "ichi",
					"2": "ni",
				},
			},
			"1": "will be overwritten",
			"2": 2,
		},
	}

	c := NewConfigurationFromMap(m)
	if !c.Exists("name", "version", "branch", "branch.key", "switches", "switches.1", "switches.2") {
		t.Error("Keys do not exists")
	} else if c.Exists("switches.__switch__") {
		t.Error("Switch still exists")
	}

	if c.String("version") != "b" {
		t.Log("Key: version")
		t.Log("Expected: b")
		t.Log("Result:", c.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c.String("branch.key") != "value" {
		t.Log("Key: branch.key")
		t.Log("Expected: value")
		t.Log("Result:", c.String("branch.key"))
		t.Error("Expected value and result does not match")
	}
	if c.Int("branch.int") != 42 {
		t.Log("Key: branch.int")
		t.Log("Expected: 42")
		t.Log("Result:", c.Int("branch.int"))
		t.Error("Expected value and result does not match")
	}
	if c.Float64("branch.float") != 32.12 {
		t.Log("Key: branch.float")
		t.Log("Expected: 32.12")
		t.Log("Result:", c.Float64("branch.float"))
		t.Error("Expected value and result does not match")
	}
	if c.Bool("branch.bool") != true {
		t.Log("Key: branch.bool")
		t.Log("Expected: true")
		t.Log("Result:", c.Bool("branch.bool"))
		t.Error("Expected value and result does not match")
	}
	if c.String("switches.1") != "one" {
		t.Log("Key: switches.1")
		t.Log("Expected: one")
		t.Log("Result:", c.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c.String("switches.2") != "two" {
		t.Log("Key: switches.2")
		t.Log("Expected: two")
		t.Log("Result:", c.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	m["version"] = "a"
	c2 := NewConfigurationFromMap(m)

	if c2.String("version") != "a" {
		t.Log("Key: version")
		t.Log("Expected: a")
		t.Log("Result:", c2.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("switches.1") != "1" {
		t.Log("Key: switches.1")
		t.Log("Expected: 1")
		t.Log("Result:", c2.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("switches.2") != "2" {
		t.Log("Key: switches.2")
		t.Log("Expected: 2")
		t.Log("Result:", c2.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	delete(m, "version")
	c3 := NewConfigurationFromMap(m)

	if c3.Exists("version") {
		t.Error("Version still exists")
	} else if !c.Exists("version") {
		t.Error("Version gone")
	}

	if c3.String("switches.1") != "ichi" {
		t.Log("Key: switches.1")
		t.Log("Expected: ichi")
		t.Log("Result:", c3.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c3.String("switches.2") != "ni" {
		t.Log("Key: switches.2")
		t.Log("Expected: ni")
		t.Log("Result:", c3.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	c4, err := NewConfigurationFromFile("test_a.json")
	if err != nil {
		t.Error("File failed", err)
	}
	if !c4.Exists("name", "version", "branch", "branch.key", "switches", "switches.1", "switches.2") {
		t.Error("Keys do not exists")
	} else if c4.Exists("switches.__switch__") {
		t.Error("Switch still exists")
	}

	if c4.String("version") != c.String("version") {
		t.Log("Key: version")
		t.Log("Expected: b")
		t.Log("Result:", c4.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c4.String("branch.key") != c.String("branch.key") {
		t.Log("Key: branch.key")
		t.Log("Expected:", c.String("branch.key"))
		t.Log("Result:", c4.String("branch.key"))
		t.Error("Expected value and result does not match")
	}
	if c4.Int("branch.int") != c.Int("branch.int") {
		t.Log("Key: branch.int")
		t.Log("Expected:", c.Int("branch.int"))
		t.Log("Result:", c4.Int("branch.int"))
		t.Error("Expected value and result does not match")
	}
	if c4.Float64("branch.float") != c.Float64("branch.float") {
		t.Log("Key: branch.float")
		t.Log("Expected:", c.Float64("branch.float"))
		t.Log("Result:", c4.Float64("branch.float"))
		t.Error("Expected value and result does not match")
	}
	if c4.Bool("branch.bool") != c.Bool("branch.bool") {
		t.Log("Key: branch.bool")
		t.Log("Expected:", c.Bool("branch.bool"))
		t.Log("Result:", c4.Bool("branch.bool"))
		t.Error("Expected value and result does not match")
	}
	if c4.String("switches.1") != c.String("switches.1") {
		t.Log("Key: switches.1")
		t.Log("Expected:", c.String("switches.1"))
		t.Log("Result:", c4.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c4.String("switches.2") != c.String("switches.2") {
		t.Log("Key: switches.2")
		t.Log("Expected:", c.String("switches.2"))
		t.Log("Result:", c4.String("switches.2"))
		t.Error("Expected value and result does not match")
	}
}

func TestConfigInclude(t *testing.T) {
	c, err := NewConfigurationFromFile("test_b.json")
	if err != nil {
		t.Error("File failed", err)
	}
	if !c.Exists("name", "version", "branch", "branch.key", "switches", "switches.1", "switches.2") {
		t.Error("Keys do not exists")
	} else if c.Exists("switches.__switch__") {
		t.Error("Switch still exists")
	}

	if c.String("version") != "b" {
		t.Log("Key: version")
		t.Log("Expected: b")
		t.Log("Result:", c.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c.String("branch.key") != "value" {
		t.Log("Key: branch.key")
		t.Log("Expected: value")
		t.Log("Result:", c.String("branch.key"))
		t.Error("Expected value and result does not match")
	}
	if c.Int("branch.int") != 42 {
		t.Log("Key: branch.int")
		t.Log("Expected: 42")
		t.Log("Result:", c.Int("branch.int"))
		t.Error("Expected value and result does not match")
	}
	if c.Float64("branch.float") != 32.12 {
		t.Log("Key: branch.float")
		t.Log("Expected: 32.12")
		t.Log("Result:", c.Float64("branch.float"))
		t.Error("Expected value and result does not match")
	}
	if c.Bool("branch.bool") != true {
		t.Log("Key: branch.bool")
		t.Log("Expected: true")
		t.Log("Result:", c.Bool("branch.bool"))
		t.Error("Expected value and result does not match")
	}
	if c.String("switches.1") != "one" {
		t.Log("Key: switches.1")
		t.Log("Expected: one")
		t.Log("Result:", c.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c.String("switches.2") != "two" {
		t.Log("Key: switches.2")
		t.Log("Expected: two")
		t.Log("Result:", c.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	c2, err := NewConfigurationFromFile("test_d.json")
	if err != nil {
		t.Error("File failed", err)
	}
	if !c2.Exists("name", "version", "branch", "branch.key", "switches", "switches.1", "switches.2",
		"app.db_username", "app.db_password", "app.db_database") {
		t.Error("Keys do not exists")
	} else if c2.Exists("switches.__switch__") {
		t.Error("Switch still exists")
	} else if c2.Exists("app.__switch__") {
		t.Error("Switch still exists")
	}

	if c2.String("version") != "b" {
		t.Log("Key: version")
		t.Log("Expected: b")
		t.Log("Result:", c2.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("branch.key") != "value" {
		t.Log("Key: branch.key")
		t.Log("Expected: value")
		t.Log("Result:", c2.String("branch.key"))
		t.Error("Expected value and result does not match")
	}
	if c2.Int("branch.int") != 42 {
		t.Log("Key: branch.int")
		t.Log("Expected: 42")
		t.Log("Result:", c2.Int("branch.int"))
		t.Error("Expected value and result does not match")
	}
	if c2.Float64("branch.float") != 32.12 {
		t.Log("Key: branch.float")
		t.Log("Expected: 32.12")
		t.Log("Result:", c2.Float64("branch.float"))
		t.Error("Expected value and result does not match")
	}
	if c2.Bool("branch.bool") != true {
		t.Log("Key: branch.bool")
		t.Log("Expected: true")
		t.Log("Result:", c2.Bool("branch.bool"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("switches.1") != "one" {
		t.Log("Key: switches.1")
		t.Log("Expected: one")
		t.Log("Result:", c2.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("switches.2") != "two" {
		t.Log("Key: switches.2")
		t.Log("Expected: two")
		t.Log("Result:", c2.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	if c2.String("app.db_database") != "myapp" {
		t.Log("Key: app.db_database")
		t.Log("Expected: myapp")
		t.Log("Result:", c2.String("app.db_database"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("app.db_username") != "appuser" {
		t.Log("Key: app.db_username")
		t.Log("Expected: appuser")
		t.Log("Result:", c2.String("app.db_username"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("app.db_password") != "apppass" {
		t.Log("Key: app.db_password")
		t.Log("Expected: apppass")
		t.Log("Result:", c2.String("app.db_password"))
		t.Error("Expected value and result does not match")
	}

	if c2.Bool("app.ssl") != true {
		t.Log("Key: app.ssl")
		t.Log("Expected: true")
		t.Log("Result:", c2.Bool("app.ssl"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("app.domain") != "www.myapp.com" {
		t.Log("Key: app.domain")
		t.Log("Expected: www.myapp.com")
		t.Log("Result:", c2.String("app.domain"))
		t.Error("Expected value and result does not match")
	}
	if c2.Int("app.port") != 80 {
		t.Log("Key: app.port")
		t.Log("Expected: 80")
		t.Log("Result:", c2.Int("app.port"))
		t.Error("Expected value and result does not match")
	}
}

func TestRelativePathInclude(t *testing.T) {
	c, err := NewConfigurationFromFile("test/test_dir.json")
	if err != nil {
		t.Error("File failed", err)
	}
	if !c.Exists("name", "version", "branch", "branch.key", "switches", "switches.1", "switches.2",
		"app.db_username", "app.db_password", "app.db_database") {
		t.Error("Keys do not exists")
	} else if c.Exists("switches.__switch__") {
		t.Error("Switch still exists")
	} else if c.Exists("app.__switch__") {
		t.Error("Switch still exists")
	}

	if c.String("version") != "b" {
		t.Log("Key: version")
		t.Log("Expected: b")
		t.Log("Result:", c.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c.String("branch.key") != "value" {
		t.Log("Key: branch.key")
		t.Log("Expected: value")
		t.Log("Result:", c.String("branch.key"))
		t.Error("Expected value and result does not match")
	}
	if c.Int("branch.int") != 42 {
		t.Log("Key: branch.int")
		t.Log("Expected: 42")
		t.Log("Result:", c.Int("branch.int"))
		t.Error("Expected value and result does not match")
	}
	if c.Float64("branch.float") != 32.12 {
		t.Log("Key: branch.float")
		t.Log("Expected: 32.12")
		t.Log("Result:", c.Float64("branch.float"))
		t.Error("Expected value and result does not match")
	}
	if c.Bool("branch.bool") != true {
		t.Log("Key: branch.bool")
		t.Log("Expected: true")
		t.Log("Result:", c.Bool("branch.bool"))
		t.Error("Expected value and result does not match")
	}
	if c.String("switches.1") != "one" {
		t.Log("Key: switches.1")
		t.Log("Expected: one")
		t.Log("Result:", c.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c.String("switches.2") != "two" {
		t.Log("Key: switches.2")
		t.Log("Expected: two")
		t.Log("Result:", c.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	if c.String("app.db_database") != "myapp" {
		t.Log("Key: app.db_database")
		t.Log("Expected: myapp")
		t.Log("Result:", c.String("app.db_database"))
		t.Error("Expected value and result does not match")
	}
	if c.String("app.db_username") != "appuser" {
		t.Log("Key: app.db_username")
		t.Log("Expected: appuser")
		t.Log("Result:", c.String("app.db_username"))
		t.Error("Expected value and result does not match")
	}
	if c.String("app.db_password") != "apppass" {
		t.Log("Key: app.db_password")
		t.Log("Expected: apppass")
		t.Log("Result:", c.String("app.db_password"))
		t.Error("Expected value and result does not match")
	}

	if c.Bool("app.ssl") != true {
		t.Log("Key: app.ssl")
		t.Log("Expected: true")
		t.Log("Result:", c.Bool("app.ssl"))
		t.Error("Expected value and result does not match")
	}
	if c.String("app.domain") != "www.myapp.com" {
		t.Log("Key: app.domain")
		t.Log("Expected: www.myapp.com")
		t.Log("Result:", c.String("app.domain"))
		t.Error("Expected value and result does not match")
	}
	if c.Int("app.port") != 80 {
		t.Log("Key: app.port")
		t.Log("Expected: 80")
		t.Log("Result:", c.Int("app.port"))
		t.Error("Expected value and result does not match")
	}

	c2, err := NewConfigurationFromFile("test_e.json")
	if err != nil {
		t.Error("File failed", err)
	}
	if !c2.Exists("name", "version", "branch", "branch.key", "switches", "switches.1", "switches.2",
		"app.db_username", "app.db_password", "app.db_database") {
		t.Error("Keys do not exists")
	} else if c2.Exists("switches.__switch__") {
		t.Error("Switch still exists")
	} else if c2.Exists("app.__switch__") {
		t.Error("Switch still exists")
	}

	if c2.String("version") != "b" {
		t.Log("Key: version")
		t.Log("Expected: b")
		t.Log("Result:", c2.String("version"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("branch.key") != "value" {
		t.Log("Key: branch.key")
		t.Log("Expected: value")
		t.Log("Result:", c2.String("branch.key"))
		t.Error("Expected value and result does not match")
	}
	if c2.Int("branch.int") != 42 {
		t.Log("Key: branch.int")
		t.Log("Expected: 42")
		t.Log("Result:", c2.Int("branch.int"))
		t.Error("Expected value and result does not match")
	}
	if c2.Float64("branch.float") != 32.12 {
		t.Log("Key: branch.float")
		t.Log("Expected: 32.12")
		t.Log("Result:", c2.Float64("branch.float"))
		t.Error("Expected value and result does not match")
	}
	if c2.Bool("branch.bool") != true {
		t.Log("Key: branch.bool")
		t.Log("Expected: true")
		t.Log("Result:", c2.Bool("branch.bool"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("switches.1") != "one" {
		t.Log("Key: switches.1")
		t.Log("Expected: one")
		t.Log("Result:", c2.String("switches.1"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("switches.2") != "two" {
		t.Log("Key: switches.2")
		t.Log("Expected: two")
		t.Log("Result:", c2.String("switches.2"))
		t.Error("Expected value and result does not match")
	}

	if c2.String("app.db_database") != "myapp" {
		t.Log("Key: app.db_database")
		t.Log("Expected: myapp")
		t.Log("Result:", c2.String("app.db_database"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("app.db_username") != "appuser" {
		t.Log("Key: app.db_username")
		t.Log("Expected: appuser")
		t.Log("Result:", c2.String("app.db_username"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("app.db_password") != "apppass" {
		t.Log("Key: app.db_password")
		t.Log("Expected: apppass")
		t.Log("Result:", c2.String("app.db_password"))
		t.Error("Expected value and result does not match")
	}

	if c2.Bool("app.ssl") != true {
		t.Log("Key: app.ssl")
		t.Log("Expected: true")
		t.Log("Result:", c2.Bool("app.ssl"))
		t.Error("Expected value and result does not match")
	}
	if c2.String("app.domain") != "www.myapp.com" {
		t.Log("Key: app.domain")
		t.Log("Expected: www.myapp.com")
		t.Log("Result:", c2.String("app.domain"))
		t.Error("Expected value and result does not match")
	}
	if c2.Int("app.port") != 80 {
		t.Log("Key: app.port")
		t.Log("Expected: 80")
		t.Log("Result:", c2.Int("app.port"))
		t.Error("Expected value and result does not match")
	}
}
