package mvc

import (
	"encoding/base64"
	"github.com/zaolab/sunnified/web"
	"gopkg.in/mgo.v2/bson"
)

type MvcMeta [4]string
type VM map[string]interface{}

const (
	MVC_MODULE int = iota
	MVC_CONTROLLER
	MVC_ACTION
	MVC_TYPE
)

func GetMvcMeta(ctxt *web.Context) MvcMeta {
	m := MvcMeta{
		MVC_MODULE:     ctxt.Module,
		MVC_CONTROLLER: ctxt.Controller,
		MVC_ACTION:     ctxt.Action,
		MVC_TYPE:       ctxt.Ext,
	}
	if m[MVC_TYPE] == "" || m[MVC_TYPE] == "." {
		m[MVC_TYPE] = ".html"
	}
	return m
}

type Controller interface {
	Construct_(*web.Context)
	Destruct_()
}

type Mold interface {
	Cast_(string, *web.Context)
	Destroy_()
}

type BaseController struct {
	*web.Context
}

func (this *BaseController) Construct_(_ *web.Context) {}

func (this *BaseController) Destruct_() {}

type IdType interface {
	Base64() string
	Hex() string
}

type IdForeign string

func (this IdForeign) Base64() string {
	return base64.URLEncoding.EncodeToString([]byte(this))
}

func (this IdForeign) MarshalJSON() ([]byte, error) {
	return []byte(`"` + this.Base64() + `"`), nil
}

func (this IdForeign) String() string {
	return string(this)
}

func (this IdForeign) ObjectId() bson.ObjectId {
	return bson.ObjectId(this)
}

func (this IdForeign) Hex() string {
	return bson.ObjectId(this).Hex()
}

type Id struct {
	bson.ObjectId `bson:"_id"`
}

func (this Id) String() string {
	return string(this.ObjectId)
}

func (this Id) Base64() string {
	return base64.URLEncoding.EncodeToString([]byte(this.ObjectId))
}

func (this Id) Hex() string {
	return this.ObjectId.Hex()
}

func (this Id) MarshalJSON() ([]byte, error) {
	return []byte(`"` + this.Base64() + `"`), nil
}

func (this *Id) UnmarshalJSON(b []byte) (err error) {
	if count := len(b); count >= 2 {
		var dst = make([]byte, count-2)
		var n int
		if n, err = base64.URLEncoding.Decode(dst, b[1:count-1]); err == nil {
			this.ObjectId = bson.ObjectId(dst[:n])
		}
	}

	return err
}

func (this Id) IdForeign() IdForeign {
	return IdForeign(this.ObjectId)
}

func NewId() Id {
	return Id{bson.NewObjectId()}
}

func IdFromBase64(b64 string) Id {
	var dst = make([]byte, len(b64))
	if n, err := base64.URLEncoding.Decode(dst, []byte(b64)); err == nil {
		return Id{bson.ObjectId(dst[:n])}
	}
	return Id{}
}
