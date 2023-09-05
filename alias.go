// Package fastapi FastApi-Python 的Golang实现
//
// 其提供了类似于FastAPI的API设计，并提供了接口文档自动生成、请求体自动校验和返回值自动序列化等使用功能；
package fastapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/godantic"
)

//goland:noinspection GoUnusedGlobalVariable
var ( // types
	Str    = godantic.String
	String = godantic.String

	Bool    = godantic.Bool
	Boolean = godantic.Bool

	Int    = godantic.Int
	Byte   = godantic.Uint8
	Int8   = godantic.Int8
	Int16  = godantic.Int16
	Int32  = godantic.Int32
	Int64  = godantic.Int64
	Uint8  = godantic.Uint8
	Uint16 = godantic.Uint16
	Uint32 = godantic.Uint32
	Uint64 = godantic.Uint64

	Float   = godantic.Float
	Float32 = godantic.Float32
	Float64 = godantic.Float64

	List    = godantic.List
	Array   = godantic.List
	Ints    = godantic.List(godantic.Int)
	Bytes   = godantic.List(godantic.Uint8)
	Strings = godantic.List(godantic.String)
	Floats  = godantic.List(godantic.Float)
)

//goland:noinspection GoUnusedGlobalVariable
type H = map[string]any    // gin.H
type M = map[string]any    // Map
type Dict = map[string]any // python.Dict

type Ctx = Context

type SchemaIface = godantic.SchemaIface
type QueryParameter = godantic.QueryParameter
type QueryModel = godantic.QueryModel
type Field = godantic.Field
type BaseModel = godantic.BaseModel
type ValidationError = godantic.ValidationError
type HTTPValidationError = godantic.HTTPValidationError
type MetaField = godantic.MetaField
type Metadata = godantic.Metadata

type RO = Option
type Opt = Option

//goland:noinspection GoUnusedGlobalVariable
var (
	QueryJsonName     = godantic.QueryJsonName
	IsFieldRequired   = godantic.IsFieldRequired
	ReflectObjectType = godantic.ReflectObjectType
	StringsToInts     = godantic.StringsToInts
	StringsToFloats   = godantic.StringsToFloats
)

//goland:noinspection GoUnusedGlobalVariable
var (
	F = helper.CombineStrings
)
