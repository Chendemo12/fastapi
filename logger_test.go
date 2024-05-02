package fastapi

import (
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := NewDefaultLogger()

	outer := []func(args ...any){logger.Info, logger.Warn, logger.Error, logger.Debug}
	for _, o := range outer {
		for _, i := range []string{"1", "2", "3"} {
			o(i)
		}
	}
}

func TestPrint(t *testing.T) {
	type args struct {
		args []any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				args: []any{"1", "2", "3"},
			},
		},
		{
			name: "test2",
			args: args{
				args: []any{time.Now(), time.Now().Second()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Debug(tt.args.args...)
			Info(tt.args.args...)
			Warn(tt.args.args...)
			Error(tt.args.args...)
		})
	}
}

func TestPrintf(t *testing.T) {
	type args struct {
		format string
		v      []any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{
				format: "name: %s, age: %d",
				v:      []any{"lee", 18},
			},
		},
		{
			name: "test2",
			args: args{
				format: "current time is %v",
				v:      []any{time.Now()},
			},
		},
		{
			name: "test3",
			args: args{
				format: "time format is %s",
				v:      []any{time.Now().Format(time.DateTime)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Debugf(tt.args.format, tt.args.v...)
			Warnf(tt.args.format, tt.args.v...)
			Errorf(tt.args.format, tt.args.v...)
		})
	}
}
