package fastapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Chendemo12/fastapi/openapi"
	"reflect"
)

type ValidateMethod interface {
	V(obj any) error
}

type ModelBindMethod interface {
	Name() string                                 // 名称
	ContentType() string                          // MIME类型
	Validate(obj any) (err error)                 // 校验方法，对于响应首先校验，然后在 Marshal；对于请求，首先 Unmarshal 然后再校验
	Marshal(obj any) ([]byte, error)              // 序列化方法，通过 ContentType 确定响应体类型
	Unmarshal(stream []byte, obj any) (err error) // 反序列化方法，通过 "http:header:Content-Type" 推断内容类型
	New() any                                     // 创建一个新实例
}

// NothingBindMethod 空实现，用于什么也不做
type NothingBindMethod struct{}

func (m *NothingBindMethod) Name() string {
	return "NothingBindMethod"
}

func (m *NothingBindMethod) ContentType() string {
	return openapi.MIMEApplicationJSONCharsetUTF8
}

func (m *NothingBindMethod) Validate(obj any) (err error) {
	return
}

func (m *NothingBindMethod) Marshal(obj any) ([]byte, error) {
	return []byte{}, nil
}

func (m *NothingBindMethod) Unmarshal(stream []byte, obj any) (err error) {
	return
}

func (m *NothingBindMethod) New() any {
	return nil
}

type JsonBindMethod struct {
	validates []ValidateMethod
	rType     reflect.Type
}

func (m *JsonBindMethod) Name() string {
	return "JsonBindMethod"
}

func (m *JsonBindMethod) ContentType() string {
	return openapi.MIMEApplicationJSONCharsetUTF8
}

func (m *JsonBindMethod) Validate(obj any) (err error) {
	for _, validate := range m.validates {
		err = validate.V(obj)
		if err != nil {
			return err
		}
	}

	for _, f := range m.AdditionalValidates() {
		err = f(obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *JsonBindMethod) AdditionalValidates() []func(obj any) error {
	s := make([]func(obj any) error, 0)
	return s
}

func (m *JsonBindMethod) Marshal(obj any) ([]byte, error) {
	return json.Marshal(obj)
}

func (m *JsonBindMethod) Unmarshal(stream []byte, obj any) (err error) {
	err = json.Unmarshal(stream, obj)
	return
}

func (m *JsonBindMethod) New() any {
	obj := reflect.New(m.rType)
	return obj.Interface()
}

type IntegerBindMethod struct {
	JsonBindMethod
	unsigned        bool // 无符号类型
	UnsignedMaximum uint64
	UnsignedMinimum uint64
	SignedMaximum   int64
	SignedMinimum   int64
}

func (m *IntegerBindMethod) Name() string {
	return "IntegerBindMethod"
}

func (m *IntegerBindMethod) AdditionalValidates() []func(obj any) error {
	s := make([]func(obj any) error, 0)
	if m.unsigned {
		s = append(s, UnsignedIntegerMaximumV[uint64](m.UnsignedMaximum, false))
	} else {
		s = append(s, SignedIntegerMaximumV[int64](m.SignedMaximum, false))
	}

	return s
}

func (m *IntegerBindMethod) Marshal(obj any) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (m *IntegerBindMethod) Unmarshal(stream []byte, obj any) (err error) {
	//TODO implement me
	panic("implement me")
}

func (m *IntegerBindMethod) New() any {
	//TODO implement me
	panic("implement me")
}

type UnsignedInteger interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint
}

type SignedInteger interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~int
}

// UnsignedIntegerMaximumV 无符号最大值校验
func UnsignedIntegerMaximumV[T UnsignedInteger](maximum T, eq bool) func(obj any) error {
	return func(obj any) error {
		if eq && obj.(T) > maximum {
			return errors.New(fmt.Sprintf("value: %d not <= %d", obj, maximum))
		}

		if !eq && obj.(T) >= maximum {
			return errors.New(fmt.Sprintf("value: %d not < %d", obj, maximum))
		}

		return nil
	}
}

// SignedIntegerMaximumV 有符号最大值校验
func SignedIntegerMaximumV[T SignedInteger](minimum T, eq bool) func(obj any) error {
	return func(obj any) error {
		if eq && obj.(T) < minimum {
			return errors.New(fmt.Sprintf("value: %d not <= %d", obj, minimum))
		}

		if !eq && obj.(T) <= minimum {
			return errors.New(fmt.Sprintf("value: %d not < %d", obj, minimum))
		}

		return nil
	}
}
