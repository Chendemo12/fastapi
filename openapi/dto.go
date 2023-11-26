package openapi

import "reflect"

type args struct {
	field      reflect.StructField `description:"字段信息"`
	fatherType reflect.Type        `description:"父结构体类型"`
	depth      int                 `description:"层级数"`
}

func (m args) String() string {
	if m.IsAnonymousStruct() {
		return m.fatherType.String()
	}
	return m.fatherType.String()
}

func (m args) FieldType() reflect.Type {
	if m.field.Type.Kind() == reflect.Ptr {
		return m.field.Type.Elem()
	}
	return m.field.Type
}

func (m args) IsAnonymousStruct() bool {
	return isAnonymousStruct(m.field.Type)
}
