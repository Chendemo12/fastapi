package fastapi

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestQBinder_Bind(t *testing.T) {
	type fields struct {
		json jsoniter.API
	}
	type args struct {
		params map[string]any
		obj    any
	}
	tests := []struct {
		args    args
		fields  fields
		name    string
		wantErr bool
	}{
		{
			name: "number-age-unmarshal",
			fields: fields{
				json: jsoniter.ConfigCompatibleWithStandardLibrary,
			},
			args: args{
				params: map[string]any{
					"name": "lee",
					"age":  12,
				},
				obj: &Person{},
			},
			wantErr: false,
		},
		{
			name: "string-age-unmarshal",
			fields: fields{
				json: jsoniter.ConfigCompatibleWithStandardLibrary,
			},
			args: args{
				params: map[string]any{
					"name": "lee",
					"age":  "123",
				},
				obj: &Person{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := StructQueryBind{json: tt.fields.json}
			if err := b.Unmarshal(tt.args.params, tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
