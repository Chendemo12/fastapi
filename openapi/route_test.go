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
