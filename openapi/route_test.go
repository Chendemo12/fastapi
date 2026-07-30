package openapi

import (
	"reflect"
	"testing"
)

func TestParseGenericModelName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "1",
			args: args{
				name: "test.PageResp[[]*github.com/Chendemo12/fastapi/test.MemoryNote]",
			},
			want: []string{"test.PageResp", "[]*github.com/Chendemo12/fastapi/test.MemoryNote"},
		},
		{
			name: "2",
			args: args{
				name: "*test.PageResp[*test.MemoryNote]",
			},
			want: []string{"*test.PageResp", "*test.MemoryNote"},
		},
		{
			name: "3",
			args: args{
				name: "test.PageResp[int]",
			},
			want: []string{"test.PageResp", "int"},
		},
		{
			name: "4",
			args: args{
				name: "test.PageResp[Sheet[int]]",
			},
			want: []string{"test.PageResp", "Sheet", "int"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseGenericModelName(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseGenericModelName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 模拟 model.JsonData 场景: 具名切片类型作为结构体字段
type testBenefitItem struct {
	Key   string  `json:"key"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type testJsonData []*testBenefitItem

// 包含具名切片指针字段的结构体（模拟 schema.BenefitCreateReq）
type testCreateReq struct {
	Name     string        `json:"name" validate:"required"`
	JsonData *testJsonData `json:"jsonData" validate:"required"`
}

// 包含具名切片字段的结构体（非指针）
type testCreateReqNoPtr struct {
	Name     string       `json:"name"`
	JsonData testJsonData `json:"jsonData"`
}

// 具名切片作为顶层响应类型
type testNamedSliceResponse []*testBenefitItem

func TestNamedSliceTypeInStruct(t *testing.T) {
	t.Run("pointer to named slice in struct", func(t *testing.T) {
		req := &testCreateReq{}
		_, err := BaseModelMetaFrom(req, 0, RouteParamRequest)
		if err != nil {
			t.Errorf("BaseModelMetaFrom with *NamedSlice should not error, got: %v", err)
		}
	})

	t.Run("named slice value in struct", func(t *testing.T) {
		req := &testCreateReqNoPtr{}
		_, err := BaseModelMetaFrom(req, 0, RouteParamRequest)
		if err != nil {
			t.Errorf("BaseModelMetaFrom with NamedSlice value should not error, got: %v", err)
		}
	})

	t.Run("named slice as response", func(t *testing.T) {
		resp := &testNamedSliceResponse{}
		_, err := BaseModelMetaFrom(resp, 0, RouteParamResponse)
		if err != nil {
			t.Errorf("BaseModelMetaFrom with NamedSlice response should not error, got: %v", err)
		}
	})

	t.Run("plain slice of structs", func(t *testing.T) {
		req := &struct {
			Items []*testBenefitItem `json:"items"`
		}{}
		_, err := BaseModelMetaFrom(req, 0, RouteParamRequest)
		if err != nil {
			t.Errorf("BaseModelMetaFrom with plain slice should not error, got: %v", err)
		}
	})
}
