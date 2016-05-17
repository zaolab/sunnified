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

func (am *ActionMeta) Name() string {
	return am.name
}

func (am *ActionMeta) RMeth() reflect.Method {
	return am.rmeth
}

func (am *ActionMeta) ReqMeth() ReqMethod {
	return am.reqmeth
}

func (am *ActionMeta) Args() []*ArgMeta {
	out := make([]*ArgMeta, len(am.args))
	copy(out, am.args)
	return out
}

func (rs ResultStyle) IsNil() bool {
	return !rs.view && !rs.vmap && !rs.mapsi
}

func (rs ResultStyle) View() bool {
	return rs.view
}

func (rs ResultStyle) Vmap() bool {
	return rs.vmap
}

func (rs ResultStyle) MapSI() bool {
	return rs.mapsi
}

func (rs ResultStyle) Status() bool {
	return rs.status
}

func (dm *DataMeta) Name() string {
	return dm.name
}

func (dm *DataMeta) LName() string {
	return dm.lname
}

func (dm *DataMeta) T() DataType {
	return dm.t
}

func (dm *DataMeta) RType() reflect.Type {
	return dm.rtype
}

func (dm *DataMeta) Fields() []*FieldMeta {
	out := make([]*FieldMeta, len(dm.fields))
	copy(out, dm.fields)
	return out
}

func (fm *FieldMeta) Rexexp() *regexp.Regexp {
	return fm.rex
}

func (fm *FieldMeta) Anonymous() bool {
	return fm.anonymous
}

func (fm *FieldMeta) Tag() reflect.StructTag {
	return fm.tag
}
