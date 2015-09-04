package controller

import (
	"reflect"
	"regexp"
)

type DataType int

const (
	DATATYPE_STRING_SUFFIX   = "_String"
	DATATYPE_INT_SUFFIX      = "_Int"
	DATATYPE_INT64_SUFFIX    = "_Int64"
	DATATYPE_FLOAT_SUFFIX    = "_Float"
	DATATYPE_FLOAT64_SUFFIX  = "_Float64"
	DATATYPE_EMAIL_SUFFIX    = "_Email"
	DATATYPE_URL_SUFFIX      = "_Url"
	DATATYPE_DATE_SUFFIX     = "_Date"
	DATATYPE_TIME_SUFFIX     = "_Time"
	DATATYPE_DATETIME_SUFFIX = "_Datetime"
	DATATYPE_BOOL_SUFFIX     = "_Bool"

	FORM_VALUETYPE_TAG_NAME = "value.type"
	FORM_VALUETYPE_LPREFIX  = "form_"
)

const (
	DATATYPE_WEBCONTEXT DataType = 1 + iota
	DATATYPE_REQUEST
	DATATYPE_RESPONSEWRITER
	DATATYPE_UPATH
	DATATYPE_UPATH_SLICE
	DATATYPE_PDATA
	DATATYPE_PDATA_MAP
	DATATYPE_STRUCT
	DATATYPE_STRING
	DATATYPE_INT
	DATATYPE_INT64
	DATATYPE_FLOAT
	DATATYPE_FLOAT64
	DATATYPE_EMAIL
	DATATYPE_URL
	DATATYPE_DATE
	DATATYPE_TIME
	DATATYPE_DATETIME
	DATATYPE_BOOL
	DATATYPE_EMBEDDED
)

type ActionMeta struct {
	name    string
	rmeth   reflect.Method
	reqmeth ReqMethod
	args    []*ArgMeta
	ResultStyle
}

type ResultStyle struct {
	view   bool
	vmap   bool
	mapsi  bool
	status bool
}

type ArgMeta struct {
	DataMeta
}

type FieldMeta struct {
	DataMeta
	tag       reflect.StructTag
	anonymous bool
	rex       *regexp.Regexp
}

type DataMeta struct {
	name   string
	lname  string
	t      DataType
	rtype  reflect.Type
	fields []*FieldMeta
}

func (this *ActionMeta) Name() string {
	return this.name
}

func (this *ActionMeta) RMeth() reflect.Method {
	return this.rmeth
}

func (this *ActionMeta) ReqMeth() ReqMethod {
	return this.reqmeth
}

func (this *ActionMeta) Args() []*ArgMeta {
	out := make([]*ArgMeta, len(this.args))
	copy(out, this.args)
	return out
}

func (this ResultStyle) IsNil() bool {
	return !this.view && !this.vmap && !this.mapsi
}

func (this ResultStyle) View() bool {
	return this.view
}

func (this ResultStyle) Vmap() bool {
	return this.vmap
}

func (this ResultStyle) MapSI() bool {
	return this.mapsi
}

func (this ResultStyle) Status() bool {
	return this.status
}

func (this *DataMeta) Name() string {
	return this.name
}

func (this *DataMeta) LName() string {
	return this.lname
}

func (this *DataMeta) T() DataType {
	return this.t
}

func (this *DataMeta) RType() reflect.Type {
	return this.rtype
}

func (this *DataMeta) Fields() []*FieldMeta {
	out := make([]*FieldMeta, len(this.fields))
	copy(out, this.fields)
	return out
}

func (this *FieldMeta) Rexexp() *regexp.Regexp {
	return this.rex
}

func (this *FieldMeta) Anonymous() bool {
	return this.anonymous
}

func (this *FieldMeta) Tag() reflect.StructTag {
	return this.tag
}
