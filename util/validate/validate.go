package validate

import (
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	email *regexp.Regexp = regexp.MustCompile("^[a-zA-Z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-zA-Z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?\\.)+[a-zA-Z0-9](?:[a-zA-Z0-9-]*[a-zA-Z0-9])?$")
	vurl  *regexp.Regexp = regexp.MustCompile(`^((?:ftp|http|https):\/\/)?(?:[\w\.\-\+]+:{0,1}[\w\.\-\+]*@)?(?:[a-z0-9\-\.]+)(?::[0-9]+)?(?:\/|\/(?:[\w#!:\.\?\+=&%@!\-\/\(\)]+)|\?(?:[\w#!:\.\?\+=&%@!\-\/\(\)]+))?$`)
	jsonp *regexp.Regexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_\.]*$`)
)

var validatormap = map[string]interface{}{
	"isemail":         IsEmail,
	"isurl":           IsUrl,
	"isjsonpcallback": IsJSONPCallback,
	"isnotempty":      IsNotEmpty,
	"isin":            IsIn,
}

func IsEmail(s string) bool {
	return email.MatchString(s)
}

func IsUrl(s string) bool {
	return vurl.MatchString(s)
}

func IsJSONPCallback(s string) bool {
	return jsonp.MatchString(s)
}

func IsNotEmpty(s string) bool {
	return strings.TrimSpace(s) != ""
}

func IsIn(s string, in ...string) bool {
	for _, v := range in {
		if v == s {
			return true
		}
	}

	return false
}

type ReqValidator bool

func (this *ReqValidator) Validate(m interface{}, req url.Values) bool {
	defer func() {
		if err := recover(); err != nil {
			*this = false
		}
	}()

	mod := reflect.ValueOf(m)
	mod = reflect.Indirect(mod.Elem())

	if mod.Kind() == reflect.Struct {
		modtyp := mod.Type()

		for i := 0; i < mod.NumField(); i++ {
			field := mod.Field(i)
			fieldtyp := modtyp.Field(i)
			lname := strings.ToLower(fieldtyp.Name)

			if _, exists := req[lname]; !exists && strings.Contains(string(fieldtyp.Tag), "va_default:") {
				SetValue(field, fieldtyp.Tag.Get("va_default"))
				continue
			}

			vfuncstr := strings.ToLower(fieldtyp.Tag.Get("va_func"))

			if vfuncstr == "pass" {
				if SetValue(field, req.Get(lname)) {
					continue
				} else {
					*this = false
					return false
				}
			} else {
				vfunc, exists := validatormap[vfuncstr]
				val := req.Get(lname)

				if !field.CanSet() || !exists {
					continue
				}

				vargs := fieldtyp.Tag.Get("va_args")
				var pass = false

				if vargs == "" {
					pass = vfunc.(func(string) bool)(val)
				} else {
					args := strings.Split(vargs, ";")
					argslen := len(args)
					values := make([]reflect.Value, argslen+1)
					values[0] = reflect.ValueOf(val)

					for i := 1; i < argslen; i++ {
						values[i] = reflect.ValueOf(args[i-1])
					}

					result := reflect.ValueOf(vfunc).Call(values)
					pass = result[0].Bool()
				}

				if pass {
					SetValue(field, val)
				} else {
					*this = false
					return false
				}
			}
		}
	}

	*this = true
	return true
}

func SetValue(field reflect.Value, val string) (ok bool) {
	defer func() {
		if err := recover(); err != nil {
			ok = false
		}
	}()

	ok = true

	switch field.Kind() {
	case reflect.String:
		field.SetString(val)
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		i, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			field.SetInt(i)
		}
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		ui, err := strconv.ParseUint(val, 10, 64)
		if err == nil {
			field.SetUint(ui)
		}
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		fl, err := strconv.ParseFloat(val, 64)
		if err == nil {
			field.SetFloat(fl)
		}
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err == nil {
			field.SetBool(b)
		}
	}

	return
}
