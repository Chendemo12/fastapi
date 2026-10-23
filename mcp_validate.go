package fastapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/Chendemo12/fastapi/utils"
	"github.com/go-playground/validator/v10"
)

const (
	MCPValidateLangEN = "en"
	MCPValidateLangZH = "zh"
)

var mcpValidator = newMCPValidator()
var mcpValidateLang = MCPValidateLangEN

// newMCPValidator 构建 MCP 专用校验器，字段名使用 json tag。
func newMCPValidator() *validator.Validate {
	v := validator.New()
	v.SetTagName("validate")
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		return utils.QueryJsonName(field.Tag, field.Name)
	})
	return v
}

// SetMCPValidateLang 配置 MCP 校验错误的语言，支持 "zh"（中文）与 "en"（英文，默认）。
func SetMCPValidateLang(lang string) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh", "zh-cn", "zh_cn", "cn":
		mcpValidateLang = MCPValidateLangZH
	default:
		mcpValidateLang = MCPValidateLangEN
	}
}

// validateMCPStruct 校验单个结构体入参（含指针），校验失败时返回带语言提示的错误。
func validateMCPStruct(v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	return translateMCPValidatorError(mcpValidator.Struct(v.Interface()))
}

func translateMCPValidatorError(err error) error {
	if err == nil {
		return nil
	}
	var ves validator.ValidationErrors
	if errors.As(err, &ves) && len(ves) > 0 {
		fe := ves[0]
		return fmt.Errorf("%s %s", fe.Field(), mcpValidationMessage(fe))
	}
	return err
}

func mcpValidationMessage(fe validator.FieldError) string {
	if mcpValidateLang != MCPValidateLangZH {
		return fe.Error()
	}

	switch fe.Tag() {
	case "required":
		return "不能为空"
	case "lte":
		return fmt.Sprintf("不能大于 %s", fe.Param())
	case "gte":
		return fmt.Sprintf("不能小于 %s", fe.Param())
	case "min":
		return fmt.Sprintf("不能小于 %s", fe.Param())
	case "max":
		return fmt.Sprintf("不能大于 %s", fe.Param())
	case "len":
		return fmt.Sprintf("长度必须等于 %s", fe.Param())
	case "oneof":
		return fmt.Sprintf("只能是以下值之一：%s", strings.ReplaceAll(fe.Param(), " ", ", "))
	case "datetime":
		return fmt.Sprintf("格式应为 %s", fe.Param())
	default:
		return fe.Error()
	}
}
