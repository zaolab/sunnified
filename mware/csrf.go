package mware

import (
	"fmt"
	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/sec"
	"github.com/zaolab/sunnified/web"
	"html"
	"html/template"
	"net/http"
	"reflect"
)

const d_securityKey = "\x98\x1e\x08\xcc\x38\x87\xbc\xe0\x48\x2f\xac\x76\x99\xc4\x9e\xbd\x72\x12\xf5\x55\xe7\x1f\x43\x74\x06\x9d\x1b\xf0\x93\x4e\xc5\x54"
const d_csrfToken = "\xc7\xeb\x58\x79\xa7\xf2\x15\x54\x06\x34\x24\x52\x50\x33\x0f\x4b\x95\x36\xb0\xb7\xdb\x5d\xa7\x07\xcf\xa5\x1c\xa5\x10\xe7\xd4\x45"

var csrfgate = sec.NewCSRFGate(sec.CSRFGateConfig{Key: []byte(d_securityKey), Token: []byte(d_csrfToken)})
var type_csrfveri = reflect.TypeOf((CsrfCheck(false)))

func init() {
	mvc.AddFuncName("URLWToken")
	mvc.AddFuncName("csrftoken_value")
	mvc.AddFuncName("csrftoken_name")
	mvc.AddFuncName("csrftoken_formtoken")
}

type CsrfCheck bool

type CsrfMiddleWare struct {
	BaseMiddleWare
}

func NewCsrfMiddleWare() CsrfMiddleWare {
	return CsrfMiddleWare{}
}

func CsrfMiddleWareConstructor() MiddleWare {
	return NewCsrfMiddleWare()
}

type CsrfTokenGetter struct {
	context *web.Context
	token   sec.CsrfRequestBody
}

func (this *CsrfCheck) Verify(r *http.Request) (valid bool) {
	valid = csrfgate.VerifyCSRFToken(r)
	*this = CsrfCheck(valid)
	return
}

func (this CsrfMiddleWare) Controller(ctxt *web.Context, _ *controller.ControlManager) {
	token := csrfgate.CSRFToken(ctxt.Response, ctxt.Request)
	csrftoken := CsrfTokenGetter{context: ctxt, token: token}
	ctxt.SetResource("csrftoken", csrftoken)
}

func (this CsrfMiddleWare) View(ctxt *web.Context, vw mvc.View) {
	var csrftoken CsrfTokenGetter

	if ctxt.MapResourceValue("csrftoken", &csrftoken) == nil && csrftoken.context != nil {
		if dview, ok := vw.(mvc.DataView); ok {
			dview.SetData("Csrftoken_Value", csrftoken.Value())
			dview.SetData("Csrftoken_Name", csrftoken.Name())
			dview.SetData("Csrftoken_Formtoken", csrftoken.FormToken())
		}
		if fview, ok := vw.(mvc.TmplView); ok {
			fview.SetViewFunc("URLWToken", func(path string) string {
				return csrftoken.URLWToken(path)
			})
			fview.SetViewFunc("URLWTokenQ", csrftoken.URLWToken)
		}
	}
}

func (this CsrfTokenGetter) FeedStructValue(ctxt *web.Context, field *controller.FieldMeta, value reflect.Value) (reflect.Value, error) {
	if field.RType() == type_csrfveri {
		var veri CsrfCheck = false
		veri.Verify(ctxt.Request)
		value = reflect.ValueOf(veri)
	}

	return value, nil
}

func (this CsrfTokenGetter) Verify() bool {
	return csrfgate.VerifyCSRFToken(this.context.Request)
}

func (this CsrfTokenGetter) Value() string {
	return this.token.Value
}

func (this CsrfTokenGetter) Name() string {
	return this.token.Name
}

func (this CsrfTokenGetter) Cookie() *http.Cookie {
	return this.token.Cookie
}

func (this CsrfTokenGetter) Ok() bool {
	return this.token.Ok
}

func (this CsrfTokenGetter) FormToken() template.HTML {
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, html.EscapeString(this.token.Name), html.EscapeString(this.token.Value)))
}

func (this CsrfTokenGetter) URLWToken(path string, qstr ...web.Q) string {
	if qstr != nil && len(qstr) > 0 {
		qstr[0][this.token.Name] = this.token.Value
	} else {
		qstr = []web.Q{web.Q{this.token.Name: this.token.Value}}
	}

	return this.context.URL(path, qstr...)
}
