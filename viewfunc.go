package sunnified

import (
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/web"
	"html/template"
	"net/http"
	"github.com/zaolab/sunnified/util/validate"
	"strings"
	"encoding/json"
	"bytes"
	"math"
	"reflect"
)

func init() {
	mvc.AddFuncName("URLQ")
	mvc.AddFuncName("URL")
	mvc.AddFuncName("QueryStr")
	mvc.AddFuncName("TimeNow")
	mvc.AddFuncName("Request")
	mvc.AddFuncName("Nl2br")
	mvc.AddFuncName("SelectOption")
	mvc.AddFuncName("SelectMultiOption")
	mvc.AddFuncName("CheckOption")
	mvc.AddFuncName("RawHtml")
	mvc.AddFuncName("User")
	mvc.AddFuncName("Session")
	mvc.AddFuncName("Json")
	mvc.AddFuncName("Implode")
	mvc.AddFuncName("Flashes")
	mvc.AddFuncName("CropText")
	mvc.AddFuncName("IRange")
	mvc.AddFuncName("Limit")
	mvc.AddFuncName("Add")
	mvc.AddFuncName("Sub")
}

// TODO: refactor this
func setFuncMap(sunctxt *web.Context, vw mvc.View) {
	if fview, ok := vw.(mvc.TmplView); ok {
		fview.SetViewFunc("URLQ", sunctxt.URL)
		fview.SetViewFunc("URL", func(s string) string {
			return sunctxt.URL(s)
		})
		fview.SetViewFunc("Request", func() *http.Request {
			return sunctxt.Request
		})
		fview.SetViewFunc("QueryStr", sunctxt.QueryStr)
		fview.SetViewFunc("TimeNow", sunctxt.StartTime)
		fview.SetViewFunc("Nl2br", func(s string) template.HTML {
			s = strings.Replace(s, "\r\n", "\n", -1)
			s = strings.Replace(s, "\r", "\n", -1)
			return template.HTML(strings.Replace(template.HTMLEscapeString(s), "\n", "<br>\n", -1))
		})
		fview.SetViewFunc("SelectOption", func(selected ...string) template.HTMLAttr {
			if len(selected) == 2 && selected[0] == selected[1] {
				return " selected "
			}
			return ""
		})
		fview.SetViewFunc("SelectMultiOption", func(value []string, selected string) template.HTMLAttr {
			if validate.IsIn(selected, value...) {
				return " selected "
			}
			return ""
		})
		fview.SetViewFunc("CheckOption", func(selected ...string) template.HTMLAttr {
			if len(selected) == 2 && selected[0] == selected[1] {
				return " checked "
			}
			return ""
		})
		fview.SetViewFunc("RawHtml", func(s string) template.HTML {
			return template.HTML(s)
		})
		fview.SetViewFunc("User", func() interface{} {
			if sunctxt.Session != nil {
				return sunctxt.Session.AuthUser()
			}
			return nil
		})
		fview.SetViewFunc("Session", func() web.SessionManager {
			if sunctxt.Session != nil {
				return sunctxt.Session
			}
			return nil
		})
		fview.SetViewFunc("Json", func(i interface{}) template.HTML {
			b, _ := json.Marshal(i)
			return template.HTML(b)
		})
		fview.SetViewFunc("Implode", func(join string, slice []string) template.HTML {
			buf := bytes.Buffer{}
			for _, s := range slice {
				buf.WriteString(template.HTMLEscapeString(s))
				buf.WriteString(join)
			}
			if len(slice) > 0 {
				buf.Truncate(buf.Len() - len(join))
			}
			return template.HTML(buf.String())
		})
		fview.SetViewFunc("Flashes", sunctxt.AllFlashes)
		fview.SetViewFunc("CropText", func(s string, l int) string {
			if len(s) > l {
				s = s[:l-3] + "..."
			}
			return s
		})
		fview.SetViewFunc("IRange", func(i ...int) (arr []int) {
			count := len(i)
			switch count {
			case 0:
				arr = make([]int, 0)
			case 1:
				arr = make([]int, i[0]+1)
				for k := range arr {
					arr[k] = k
				}
			case 2:
				arr = make([]int, int(math.Abs(float64(i[1]-i[0])))+1)
				if i[0] > i[1] {
					for k := range arr {
						arr[k] = i[0]
						i[0]--
					}
				} else {
					for k := range arr {
						arr[k] = i[0]
						i[0]++
					}
				}
			case 3:
				if i[0] > i[1] {
					i[1] = i[1] - 1
				} else {
					i[1] = i[1] + 1
				}
				size := (float64(i[1]) - float64(i[0])) / float64(i[2])
				if size < 0 {
					return
				}
				arr = make([]int, int(math.Floor(size+0.5)))
				for k := range arr {
					arr[k] = i[0]
					i[0] = i[0] + i[2]
				}
			}
			return
		})
		fview.SetViewFunc("Limit", func(slice interface{}, limit int) interface{} {
			refslice := reflect.ValueOf(slice)
			if (refslice.Kind() == reflect.Slice || refslice.Kind() == reflect.Array) &&
			refslice.Cap() > limit {
				slice = refslice.Slice(0, limit).Interface()
			}
			return slice
		})
		fview.SetViewFunc("Add", func(num1 int, num2 int) int {
			return num1 + num2
		})
		fview.SetViewFunc("Sub", func(num1 int, num2 int) int {
			return num1 - num2
		})
	}
}