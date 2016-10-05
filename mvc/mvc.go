package mvc

import (
	"encoding/base64"

	"github.com/zaolab/sunnified/web"
	"gopkg.in/mgo.v2/bson"
)

type Meta [4]string
type VM map[string]interface{}

const (
	MVCModule int = iota
	MVCController
	MVCAction
	MVCType
)

func GetMvcMeta(ctxt *web.Context) Meta {
	m := Meta{
		MVCModule:     ctxt.Module,
		MVCController: ctxt.Controller,
		MVCAction:     ctxt.Action,
		MVCType:       ctxt.Ext,
	}
	if m[MVCType] == "" || m[MVCType] == "." {
		m[MVCType] = ".html"
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

type IDType interface {
	Base64() string
	Hex() string
}

type IDForeign string

func (id IDForeign) Base64() string {
	return base64.URLEncoding.EncodeToString([]byte(id))
}

func (id IDForeign) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.Base64() + `"`), nil
}

func (id IDForeign) String() string {
	return string(id)
}

func (id IDForeign) ObjectId() bson.ObjectId {
	return bson.ObjectId(id)
}

func (id IDForeign) Hex() string {
	return bson.ObjectId(id).Hex()
}

type ID struct {
	bson.ObjectId `bson:"_id"`
}

func (id ID) String() string {
	return string(id.ObjectId)
}

func (id ID) Base64() string {
	return base64.URLEncoding.EncodeToString([]byte(id.ObjectId))
}

func (id ID) Hex() string {
	return id.ObjectId.Hex()
}

func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.Base64() + `"`), nil
}

func (id *ID) UnmarshalJSON(b []byte) (err error) {
	if count := len(b); count >= 2 {
		var dst = make([]byte, count-2)
		var n int
		if n, err = base64.URLEncoding.Decode(dst, b[1:count-1]); err == nil {
			id.ObjectId = bson.ObjectId(dst[:n])
		}
	}

	return err
}

func (id ID) IDForeign() IDForeign {
	return IDForeign(id.ObjectId)
}

func NewID() ID {
	return ID{bson.NewObjectId()}
}

func IDFromBase64(b64 string) ID {
	var dst = make([]byte, len(b64))
	if n, err := base64.URLEncoding.Decode(dst, []byte(b64)); err == nil {
		return ID{bson.ObjectId(dst[:n])}
	}
	return ID{}
}
