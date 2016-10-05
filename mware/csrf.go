package mware

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"reflect"

	"github.com/zaolab/sunnified/mvc"
	"github.com/zaolab/sunnified/mvc/controller"
	"github.com/zaolab/sunnified/sec"
	"github.com/zaolab/sunnified/web"
)

const dSecurityKey = "\x98\x1e\x08\xcc\x38\x87\xbc\xe0\x48\x2f\xac\x76\x99\xc4\x9e\xbd\x72\x12\xf5\x55\xe7\x1f\x43\x74\x06\x9d\x1b\xf0\x93\x4e\xc5\x54"
const dCSRFToken = "\xc7\xeb\x58\x79\xa7\xf2\x15\x54\x06\x34\x24\x52\x50\x33\x0f\x4b\x95\x36\xb0\xb7\xdb\x5d\xa7\x07\xcf\xa5\x1c\xa5\x10\xe7\xd4\x45"

var csrfgate = sec.NewCSRFGate(sec.CSRFGateConfig{Key: []byte(dSecurityKey), Token: []byte(dCSRFToken)})
var typeCSRFVeri = reflect.TypeOf((CSRFCheck(false)))

func init() {
	mvc.AddFuncName("URLWToken")
	mvc.AddFuncName("csrftoken_value")
	mvc.AddFuncName("csrftoken_name")
	mvc.AddFuncName("csrftoken_formtoken")
}

type CSRFCheck bool

type CSRFMiddleWare struct {
	BaseMiddleWare
}

func NewCSRFMiddleWare() CSRFMiddleWare {
	return CSRFMiddleWare{}
}

func CSRFMiddleWareConstructor() MiddleWare {
	return NewCSRFMiddleWare()
}

type CSRFTokenGetter struct {
	context *web.Context
	token   sec.CSRFRequestBody
}

func (cc *CSRFCheck) Verify(r *http.Request) (valid bool) {
	valid = csrfgate.VerifyCSRFToken(r)
	*cc = CSRFCheck(valid)
	return
}

func (mw CSRFMiddleWare) Controller(ctxt *web.Context, _ *controller.ControlManager) {
	token := csrfgate.CSRFToken(ctxt.Response, ctxt.Request)
	csrftoken := CSRFTokenGetter{context: ctxt, token: token}
	ctxt.SetResource("csrftoken", csrftoken)
}

func (mw CSRFMiddleWare) View(ctxt *web.Context, vw mvc.View) {
	var csrftoken CSRFTokenGetter

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

func (ct CSRFTokenGetter) FeedStructValue(ctxt *web.Context,
	field *controller.FieldMeta,
	value reflect.Value) (reflect.Value, error) {

	if field.RType() == typeCSRFVeri {
		var veri CSRFCheck = false
		veri.Verify(ctxt.Request)
		value = reflect.ValueOf(veri)
	}

	return value, nil
}

func (ct CSRFTokenGetter) Verify() bool {
	return csrfgate.VerifyCSRFToken(ct.context.Request)
}

func (ct CSRFTokenGetter) Value() string {
	return ct.token.Value
}

func (ct CSRFTokenGetter) Name() string {
	return ct.token.Name
}

func (ct CSRFTokenGetter) Cookie() *http.Cookie {
	return ct.token.Cookie
}

func (ct CSRFTokenGetter) Ok() bool {
	return ct.token.Ok
}

func (ct CSRFTokenGetter) FormToken() template.HTML {
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`,
		html.EscapeString(ct.token.Name), html.EscapeString(ct.token.Value)))
}

func (ct CSRFTokenGetter) URLWToken(path string, qstr ...web.Q) string {
	if qstr != nil && len(qstr) > 0 {
		qstr[0][ct.token.Name] = ct.token.Value
	} else {
		qstr = []web.Q{web.Q{ct.token.Name: ct.token.Value}}
	}

	return ct.context.URL(path, qstr...)
}
