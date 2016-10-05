package controller

import (
	"reflect"
	"regexp"
)

type DataType int

const (
	DatatypeStringSuffix = "_String"
	DatatypeIntSuffix = "_Int"
	DatatypeInt64Suffix = "_Int64"
	DatatypeFloatSuffix = "_Float"
	DatatypeFloat64Suffix = "_Float64"
	DatatypeEmailSuffix = "_Email"
	DatatypeURLSuffix = "_Url"
	DatatypeDateSuffix = "_Date"
	DatatypeTimeSuffix = "_Time"
	DatatypeDateTimeSuffix = "_Datetime"
	DatatypeBoolSuffix = "_Bool"

	FormValueTypeTagName = "value.type"
	FormValueTypeLprefix = "form_"
)

const (
	DatatypeWebContext DataType = 1 + iota
	DatatypeRequest
	DatatypeResponseWriter
	DatatypeUpath
	DatatypeUpathSlice
	DatatypePdata
	DatatypePdataMap
	DatatypeStruct
	DatatypeString
	DatatypeInt
	DatatypeInt64
	DatatypeFloat
	DatatypeFloat64
	DatatypeEmail
	DatatypeURL
	DatatypeDate
	DatatypeTime
	DatatypeDateTime
	DatatypeBool
	DatatypeEmbedded
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
