package fastapi

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// TestParseJsoniterErrorNotPanic 覆盖历史缺陷：非 jsoniter 错误文本曾导致下标越界 panic。
func TestParseJsoniterErrorNotPanic(t *testing.T) {
	type jsonErr struct{ msg string }
	errs := []error{
		errors.New("plain error message"),
		errors.New("json: cannot unmarshal string into Go struct field Req.size of type int"),
		errors.New("Mismatch type int with value string \"at index 5: mismatched type\n\n\t{\"size\":\"3\"}"),
		errors.New("only|one|separator"),
		errors.New("a|b|c|d|e"), // 段数足够但不是合法 JSON
		&json.UnmarshalTypeError{Value: "string", Type: reflect.TypeOf(int64(0)), Field: "size"},
	}

	for _, err := range errs {
		ve := ParseJsoniterError(err, "query", "")
		if ve == nil {
			t.Fatalf("err=%q: nil ValidationError", err.Error())
		}
		if ve.Msg == "" {
			t.Fatalf("err=%q: empty Msg", err.Error())
		}
		if len(ve.Loc) == 0 || ve.Loc[0] != "query" {
			t.Fatalf("err=%q: unexpected loc %v", err.Error(), ve.Loc)
		}
		if ve.Ctx == nil {
			t.Fatalf("err=%q: nil Ctx", err.Error())
		}
	}
}

// TestParseJsoniterErrorJsoniterFormat 校验 jsoniter 格式文本仍可解析
func TestParseJsoniterErrorJsoniterFormat(t *testing.T) {
	err := errors.New(`main.Req.size: ReadString: expects " or n, but found 3, error found in #10 byte of ...|"size": 3|..., bigger context ...|{"size":3}|...`)

	ve := ParseJsoniterError(err, "query", "Req")
	if ve == nil {
		t.Fatal("nil ValidationError")
	}
	if ve.Msg == "" {
		t.Fatal("empty Msg")
	}
	if ve.Ctx[whereErrorLabel] != "client" {
		t.Fatalf("unexpected ctx: %v", ve.Ctx)
	}
}

// TestParseJsoniterErrorCtxIsolation 校验错误上下文不再复用包级 map（避免跨请求污染与 data race）
func TestParseJsoniterErrorCtxIsolation(t *testing.T) {
	first := ParseJsoniterError(errors.New("first error"), "query", "")
	second := ParseJsoniterError(errors.New("second error"), "query", "")

	first.Ctx[modelDescLabel] = "FirstModel"

	if _, ok := second.Ctx[modelDescLabel]; ok {
		t.Fatal("Ctx 被复用：第二个错误的上下文被第一个错误的写入污染")
	}
	if _, ok := whereClientError[modelDescLabel]; ok {
		t.Fatal("包级 whereClientError 被污染")
	}
}

// TestParseValidatorErrorCtxIsolation 校验 validator 错误上下文同样相互隔离
func TestParseValidatorErrorCtxIsolation(t *testing.T) {
	type sample struct {
		Name string `json:"name" validate:"required"`
	}

	LazyInit()
	err := defaultValidator.Struct(&sample{})
	ves := ParseValidatorError(err, "request", "Sample")
	if len(ves) == 0 {
		t.Fatal("expected validation errors")
	}

	ves[0].Ctx[modelDescLabel] = "SampleModel"

	ves2 := ParseValidatorError(err, "request", "Other")
	if _, ok := ves2[0].Ctx[modelDescLabel]; ok {
		t.Fatal("Ctx 被复用：validator 错误的上下文相互污染")
	}
	if _, ok := whereClientError[modelDescLabel]; ok {
		t.Fatal("包级 whereClientError 被污染")
	}
}
