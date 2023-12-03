// Package fastapi FastApi-Python 的Golang实现
//
// 其提供了类似于FastAPI的API设计，并提供了接口文档自动生成、请求体自动校验和返回值自动序列化等使用功能；
package fastapi

import (
	"github.com/Chendemo12/fastapi-tool/helper"
	"github.com/Chendemo12/fastapi/utils"
)

//goland:noinspection GoUnusedGlobalVariable
type H = map[string]any    // gin.H
type Dict = map[string]any // python.Dict

type Ctx = Context

type Opt = Option

//goland:noinspection GoUnusedGlobalVariable
var (
	ReflectObjectType = utils.ReflectObjectType
)

//goland:noinspection GoUnusedGlobalVariable
var (
	F = helper.CombineStrings
)
