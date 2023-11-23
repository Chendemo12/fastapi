package openapi

import "reflect"

// AnonymousModelNameConnector 为匿名结构体生成一个名称, 连接符
const AnonymousModelNameConnector = "_"

// Deprecated: BaseModelToMetadata 提取基本数据模型的元信息
//
//	@param	model	SchemaIface	基本数据模型
//	@return	*Metadata 基本数据模型的元信息
func BaseModelToMetadata(model SchemaIface) *Metadata {
	if model == nil { // 冗余校验
		return nil
	}
	if md, ok := model.(*Metadata); ok { // 接口处定义了基本数据类型和List
		return md
	}

	meta := &Metadata{}
	meta.FromModel(model)

	metaCache.Set(meta)
	return meta
}

// ----------------------------------------------------------------------------

// 缓存全部的结构体元信息，以减少上层反射次数, 仅用于通过 BaseModel 获取 Metadata
var metaCache = &MetaCache{data: make([]*Metadata, 0)}

// GetMetadata 获取结构体的元信息
func GetMetadata(pkg string) *Metadata { return metaCache.Get(pkg) }

// MetaCache Metadata 缓存
type MetaCache struct {
	data []*Metadata
}

func (m *MetaCache) Get(pkg string) *Metadata {
	for i := 0; i < len(m.data); i++ {
		if m.data[i].String() == pkg {
			return m.data[i]
		}
	}
	return nil
}

// Set 保存一个元信息，存在则更新
func (m *MetaCache) Set(meta *Metadata) {
	for i := 0; i < len(m.data); i++ {
		if m.data[i].String() == meta.String() {
			m.data[i] = meta
			return
		}
	}
	m.data = append(m.data, meta)
}

// ----------------------------------------------------------------------------

// 是否是匿名(未声明)的结构体
func isAnonymousStruct(fieldType reflect.Type) bool {
	if fieldType.Kind() == reflect.Ptr {
		return fieldType.Elem().Name() == ""
	}
	return fieldType.Name() == ""
}

func getReflectType(rt reflect.Type) reflect.Type {
	var fieldType reflect.Type

	switch rt.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		fieldType = rt.Elem()
	default:
		fieldType = rt
	}

	return fieldType
}

func getModelNames(fieldMeta *MetaField, fieldType reflect.Type) (string, string) {

	var pkg, name string
	if isAnonymousStruct(fieldType) {
		// 未命名的结构体类型, 没有名称, 分配包名和名称
		name = fieldMeta.Title + "Model"
		//pkg = fieldMeta._pkg + AnonymousModelNameConnector + name
		pkg = fieldMeta._pkg
	} else {
		pkg = fieldType.String() // 关联模型
		name = fieldType.Name()
	}

	return pkg, name
}
