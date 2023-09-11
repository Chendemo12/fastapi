// Package fastapi FastApi-Python 的Golang实现
//
// 其提供了类似于FastAPI的API设计，并提供了接口文档自动生成、请求体自动校验和返回值自动序列化等使用功能；
package fastapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/openapi"
)

//goland:noinspection GoUnusedGlobalVariable
var ( // types
	Str    = openapi.String
	String = openapi.String

	Bool    = openapi.Bool
	Boolean = openapi.Bool

	Int    = openapi.Int
	Byte   = openapi.Uint8
	Int8   = openapi.Int8
	Int16  = openapi.Int16
	Int32  = openapi.Int32
	Int64  = openapi.Int64
	Uint8  = openapi.Uint8
	Uint16 = openapi.Uint16
	Uint32 = openapi.Uint32
	Uint64 = openapi.Uint64

	Float   = openapi.Float
	Float32 = openapi.Float32
	Float64 = openapi.Float64

	List    = openapi.List
	Array   = openapi.List
	Ints    = openapi.List(openapi.Int)
	Bytes   = openapi.List(openapi.Uint8)
	Strings = openapi.List(openapi.String)
	Floats  = openapi.List(openapi.Float)
)

//goland:noinspection GoUnusedGlobalVariable
type H = map[string]any    // gin.H
type M = map[string]any    // Map
type Dict = map[string]any // python.Dict

type Ctx = Context

type SchemaIface = openapi.SchemaIface
type QueryParameter = openapi.QueryParameter
type QueryModel = openapi.QueryModel
type Field = openapi.Field
type BaseModel = openapi.BaseModel
type ValidtionError = openapi.ValidationError
type HTTPValidationError = openapi.HTTPValidationError
type MetaField = openapi.MetaField
type Metadata = openapi.Metadata

type RO = Option
type Opt = Option

//goland:noinspection GoUnusedGlobalVariable
var (
	QueryJsonName     = openapi.QueryJsonName
	IsFieldRequired   = openapi.IsFieldRequired
	ReflectObjectType = openapi.ReflectObjectType
	StringsToInts     = openapi.StringsToInts
	StringsToFloats   = openapi.StringsToFloats
)

//goland:noinspection GoUnusedGlobalVariable
var (
	F = helper.CombineStrings
)
