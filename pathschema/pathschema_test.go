package pathschema

import (
	"reflect"
	"testing"
)

func TestBackslash_Connector(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "/",
			want: "/",
		},
		{
			name: "/",
			want: "/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Backslash{}
			if got := s.Connector(); got != tt.want {
				t.Errorf("Connector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackslash_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Backslash",
			want: "Backslash",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Backslash{}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackslash_Split(t *testing.T) {
	type args struct {
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ReadProcTree",
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"Read", "Proc", "Tree"},
		},
		{
			name: "File2Dir",
			args: args{
				relativePath: "File2Dir",
			},
			want: []string{"File2", "Dir"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Backslash{}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowerCamelCase_Connector(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LowerCamelCase{}
			if got := s.Connector(); got != tt.want {
				t.Errorf("Connector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowerCamelCase_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "LowerCamelCase",
			want: "LowerCamelCase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LowerCamelCase{}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowerCamelCase_Split(t *testing.T) {
	type args struct {
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ReadProcTree",
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"readProcTree"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LowerCamelCase{}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowerCase_Connector(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LowerCase{}
			if got := s.Connector(); got != tt.want {
				t.Errorf("Connector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowerCase_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "LowerCase",
			want: "LowerCase",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LowerCase{}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowerCase_Split(t *testing.T) {
	type args struct {
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ReadProcTree",
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"read", "proc", "tree"},
		},
		{
			name: "Read3Proc2Tree",
			args: args{
				relativePath: "Read3Proc2Tree",
			},
			want: []string{"read3", "proc2", "tree"},
		},
		{
			name: "Read333Proc222Tree11",
			args: args{
				relativePath: "Read333Proc222Tree11",
			},
			want: []string{"read333", "proc222", "tree11"},
		},
		{
			name: "Read3-3Proc2-2Tree-1",
			args: args{
				relativePath: "Read3-3Proc2-2Tree-1",
			},
			want: []string{"read3-3", "proc2-2", "tree-1"},
		},
		{
			name: "Read_-3Proc-_2Tree_1",
			args: args{
				relativePath: "Read_-3Proc-_2Tree_1",
			},
			want: []string{"read_-3", "proc-_2", "tree_1"},
		},
		{
			name: "123456",
			args: args{
				relativePath: "123456",
			},
			want: []string{"123456"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := LowerCase{}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOriginal_Connector(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Original{}
			if got := s.Connector(); got != tt.want {
				t.Errorf("Connector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOriginal_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Original",
			want: "Original",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Original{}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOriginal_Split(t *testing.T) {
	type args struct {
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ReadProcTree",
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"ReadProcTree"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Original{}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnderline_Connector(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "_",
			want: "_",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Underline{}
			if got := s.Connector(); got != tt.want {
				t.Errorf("Connector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnderline_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "Underline",
			want: "Underline",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Underline{}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnderline_Split(t *testing.T) {
	type args struct {
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ReadProcTree",
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"Read", "Proc", "Tree"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Underline{}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnixDash_Connector(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "-",
			want: "-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UnixDash{}
			if got := s.Connector(); got != tt.want {
				t.Errorf("Connector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnixDash_Name(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "UnixDash",
			want: "UnixDash",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UnixDash{}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnixDash_Split(t *testing.T) {
	type args struct {
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ReadProcTree",
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"Read", "Proc", "Tree"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := UnixDash{}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComposition_Name(t *testing.T) {
	type fields struct {
		linker  string
		schemas []RoutePathSchema
	}
	tests := []struct {
		name   string
		want   string
		fields fields
	}{
		{
			name:   "Composition",
			fields: fields{},
			want:   "Composition",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Composition{
				schemas: tt.fields.schemas,
				linker:  tt.fields.linker,
			}
			if got := s.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComposition_Split(t *testing.T) {
	type fields struct {
		linker  string
		schemas []RoutePathSchema
	}
	type args struct {
		relativePath string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "ReadProcTree",
			fields: fields{
				schemas: []RoutePathSchema{&LowerCase{}, &UnixDash{}},
				linker:  "-",
			},
			args: args{
				relativePath: "ReadProcTree",
			},
			want: []string{"read", "proc", "tree"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Composition{
				schemas: tt.fields.schemas,
				linker:  tt.fields.linker,
			}
			if got := s.Split(tt.args.relativePath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Split() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUppercaseFirstLetter(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "upperPrice",
			args: args{
				s: "upperPrice",
			},
			want: "UpperPrice",
		},
		{
			name: "HaJiMi",
			args: args{
				s: "HaJiMi",
			},
			want: "HaJiMi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UppercaseFirstLetter(tt.args.s); got != tt.want {
				t.Errorf("UppercaseFirstLetter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitWords(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "IsPreventCheat",
			args: args{
				s: "IsPreventCheat",
			},
			want: []string{"Is", "Prevent", "Cheat"},
		},
		{
			name: "HaJiMi",
			args: args{
				s: "HaJiMi",
			},
			want: []string{"Ha", "Ji", "Mi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitWords(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitWords() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLowercaseFirstLetter(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "LowerPrice",
			args: args{
				s: "LowerPrice",
			},
			want: "lowerPrice",
		},
		{
			name: "HaJiMi",
			args: args{
				s: "HaJiMi",
			},
			want: "haJiMi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LowercaseFirstLetter(tt.args.s); got != tt.want {
				t.Errorf("LowercaseFirstLetter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	type args struct {
		schema       RoutePathSchema
		prefix       string
		relativePath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "UnixDash",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       &UnixDash{},
			},
			want: "/api/Read-Unix-Proc-Tree",
		},
		{
			name: "Underline",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       &Underline{},
			},
			want: "/api/Read_Unix_Proc_Tree",
		},
		{
			name: "LowerCamelCase",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       &LowerCamelCase{},
			},
			want: "/api/readUnixProcTree",
		},
		{
			name: "LowerCase",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       &LowerCase{},
			},
			want: "/api/readunixproctree",
		},
		{
			name: "Backslash",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       &Backslash{},
			},
			want: "/api/Read/Unix/Proc/Tree",
		},
		{
			name: "Original",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       &Original{},
			},
			want: "/api/ReadUnixProcTree",
		},
		{
			name: "LowerCaseDash",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       Default(),
			},
			want: "/api/read-unix-proc-tree",
		},
		{
			name: "LowerCaseUnderline",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       NewComposition(&LowerCase{}, &Underline{}),
			},
			want: "/api/read_unix_proc_tree",
		},
		{
			name: "LowerCaseDashUnderline",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       NewComposition(&LowerCase{}, &UnixDash{}, &Underline{}),
			},
			want: "/api/read-_unix-_proc-_tree",
		},
		{
			name: "LowerCaseDashUnderlineBackslash",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       NewComposition(NewComposition(&LowerCase{}, &UnixDash{}), &Underline{}, &Backslash{}),
			},
			want: "/api/read-_/unix-_/proc-_/tree",
		},
		{
			name: "LowerCaseWithPreEqual",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       NewComposition(&LowerCase{}, &AddPrefix{Prefix: "="}, &Backslash{}),
			},
			want: "/api/=read/=unix/=proc/=tree",
		},
		{
			name: "LowerCaseWithPostDash",
			args: args{
				prefix:       "/api",
				relativePath: "ReadUnixProcTree",
				schema:       NewComposition(&LowerCase{}, &AddSuffix{Suffix: "-"}, &Backslash{}),
			},
			want: "/api/read-/unix-/proc-/tree-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Format(tt.args.prefix, tt.args.relativePath, tt.args.schema); got != tt.want {
				t.Errorf("Format() = %v, want %v", got, tt.want)
			}
		})
	}
}
