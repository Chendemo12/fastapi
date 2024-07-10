package fastapi

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	end     = "\u001B[0m\t"
	fuchsia = "\u001B[35mDEBUG" // 紫红色
	blue    = "\u001B[34mINFO"
	green   = "\u001B[32mSUCC"
	red     = "\u001B[31mERROR"
	yellow  = "\u001B[33mWARN"
)

var console LoggerIface = nil

func init() {
	ReplaceLogger(NewDefaultLogger())
}

// LoggerIface 自定义logger接口，log及zap等均已实现此接口
type LoggerIface interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Warnf(format string, args ...any)
	Debugf(format string, args ...any)
}

// DefaultLogger 默认的控制台日志具柄
type DefaultLogger struct {
	info      *log.Logger // 或 logger.Printf("\t\u001B[34mINFO\u001B[0m\t%v\n", args...)
	debug     *log.Logger
	warn      *log.Logger
	error     *log.Logger
	calldepth int
}

func (l *DefaultLogger) Debug(args ...any) {
	// 获取调用者路径
	_ = l.debug.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Info(args ...any) {
	_ = l.info.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Warn(args ...any) {
	_ = l.warn.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Error(args ...any) {
	_ = l.error.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Errorf(format string, v ...any) {
	_ = l.error.Output(l.calldepth, fmt.Errorf(format, v...).Error())
}

func (l *DefaultLogger) Warnf(format string, v ...any) {
	_ = l.warn.Output(l.calldepth, fmt.Errorf(format, v...).Error())
}

func (l *DefaultLogger) Debugf(format string, v ...any) {
	_ = l.debug.Output(l.calldepth, fmt.Errorf(format, v...).Error())
}

func NewLogger(out io.Writer, prefix string, flag int) *DefaultLogger {
	d := &DefaultLogger{
		info:      log.New(out, blue+end+prefix, flag|log.Lmsgprefix),
		debug:     log.New(out, fuchsia+end+prefix, flag|log.Lmsgprefix),
		warn:      log.New(out, yellow+end+prefix, flag|log.Lmsgprefix),
		error:     log.New(out, red+end+prefix, flag|log.Lmsgprefix),
		calldepth: 2,
	}
	return d
}

func NewDefaultLogger() *DefaultLogger {
	return NewLogger(os.Stdout, "", log.LstdFlags|log.Lshortfile)
}

func ReplaceLogger(logger LoggerIface) {
	if logger == nil {
		return
	}
	console = logger

	Debug = console.Debug
	Info = console.Info
	Warn = console.Warn
	Error = console.Error
	Errorf = console.Errorf
	Warnf = console.Warnf
	Debugf = console.Debugf
}

var Info func(args ...any)
var Debug func(args ...any)
var Warn func(args ...any)
var Error func(args ...any)

var Errorf func(format string, v ...any)
var Warnf func(format string, v ...any)
var Debugf func(format string, v ...any)
