// Package fastapi FastApi-Python 的Golang实现
//
// 其提供了类似于FastAPI的API设计，并提供了接口文档自动生成、请求体自动校验和返回值自动序列化等使用功能；
package fastapi

import (
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/Chendemo12/fastapi/utils"
)

//goland:noinspection GoUnusedGlobalVariable
type H = map[string]any    // gin.H
type Dict = map[string]any // python.Dict

type Ctx = Context

type BaseRouter = BaseGroupRouter

// None 可用于POST/PATH/PUT方法的占位
type None struct{}

func (*None) SchemaDesc() string { return "Empty Request Content" }

//goland:noinspection GoUnusedGlobalVariable
var (
	ReflectObjectType = utils.ReflectObjectType
	SetJsonEngine     = utils.SetJsonEngine
)

//goland:noinspection GoUnusedGlobalVariable
var F = utils.CombineStrings

func Iter[T any, S any](seq []S, fc func(elem S) T) []T {
	ns := make([]T, len(seq))
	for i := 0; i < len(seq); i++ {
		ns[i] = fc(seq[i])
	}

	return ns
}

// SetMultiFormFileName 设置表单数据中文件的字段名（键名称）
func SetMultiFormFileName(name string) {
	openapi.MultipartFormFileName = name
}

// SetMultiFormParamName 设置表单数据中json参数的字段名（键名称）
func SetMultiFormParamName(name string) {
	openapi.MultipartFormParamName = name
}
