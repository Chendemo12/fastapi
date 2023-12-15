package fastapi

import (
	jsoniter "github.com/json-iterator/go"
	"testing"
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
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "number-age-unmarshal",
			fields: fields{
				json: queryStructJsonConf.Froze(),
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
				json: queryStructJsonConf.Froze(),
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
			b := NewStructQueryBinder("", nil)
			if err := b.Unmarshal(tt.args.params, tt.args.obj); (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}