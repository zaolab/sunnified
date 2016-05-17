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

func (c *BaseController) Construct_(_ *web.Context) {}

func (c *BaseController) Destruct_() {}

type IdType interface {
	Base64() string
	Hex() string
}

type IdForeign string

func (id IdForeign) Base64() string {
	return base64.URLEncoding.EncodeToString([]byte(id))
}

func (id IdForeign) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.Base64() + `"`), nil
}

func (id IdForeign) String() string {
	return string(id)
}

func (id IdForeign) ObjectId() bson.ObjectId {
	return bson.ObjectId(id)
}

func (id IdForeign) Hex() string {
	return bson.ObjectId(id).Hex()
}

type Id struct {
	bson.ObjectId `bson:"_id"`
}

func (id Id) String() string {
	return string(id.ObjectId)
}

func (id Id) Base64() string {
	return base64.URLEncoding.EncodeToString([]byte(id.ObjectId))
}

func (id Id) Hex() string {
	return id.ObjectId.Hex()
}

func (id Id) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.Base64() + `"`), nil
}

func (id *Id) UnmarshalJSON(b []byte) (err error) {
	if count := len(b); count >= 2 {
		var dst = make([]byte, count-2)
		var n int
		if n, err = base64.URLEncoding.Decode(dst, b[1:count-1]); err == nil {
			id.ObjectId = bson.ObjectId(dst[:n])
		}
	}

	return err
}

func (id Id) IdForeign() IdForeign {
	return IdForeign(id.ObjectId)
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
