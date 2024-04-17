package fastapi

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
)

const (
	end     = "\u001B[0m\t"
	fuchsia = "\u001B[35mDEBUG" // 紫红色
	blue    = "\u001B[34mINFO"
	green   = "\u001B[32mSUCC"
	red     = "\u001B[31mERROR"
	yellow  = "\u001B[33mWARN"
)

var console = NewDefaultLogger()

func init() {
	console.calldepth += 1
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
	isDiscard int32 // atomic boolean: whether out == io.Discard
	calldepth int
}

func (l *DefaultLogger) Debug(args ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	// 获取调用者路径
	_ = l.debug.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Info(args ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	_ = l.info.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Warn(args ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	_ = l.warn.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Error(args ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	_ = l.error.Output(l.calldepth, fmt.Sprintln(args...))
}

func (l *DefaultLogger) Errorf(format string, v ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	_ = l.error.Output(l.calldepth, fmt.Errorf(format, v...).Error())
}

func (l *DefaultLogger) Warnf(format string, v ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
	_ = l.warn.Output(l.calldepth, fmt.Errorf(format, v...).Error())
}

func (l *DefaultLogger) Debugf(format string, v ...any) {
	if atomic.LoadInt32(&l.isDiscard) != 0 {
		return
	}
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
	if out == io.Discard {
		d.isDiscard = 1
	}
	return d
}

func NewDefaultLogger() *DefaultLogger {
	return NewLogger(os.Stdout, "", log.LstdFlags|log.Lshortfile)
}

// default logger

func Debug(args ...any) { console.Debug(args...) }

func Info(args ...any) { console.Info(args...) }

func Warn(args ...any) { console.Warn(args...) }

func Error(args ...any) { console.Error(args...) }

func Errorf(format string, v ...any) { console.Errorf(format, v...) }

func Warnf(format string, v ...any) { console.Warnf(format, v...) }

func Debugf(format string, v ...any) { console.Debugf(format, v...) }
