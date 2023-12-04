package fiberEngine

import "github.com/gofiber/fiber/v2"

var recoverHandler StackTraceHandlerFunc = nil
var fiberErrorHandler fiber.ErrorHandler = nil // 设置fiber自定义错误处理函数

// StackTraceHandlerFunc 错误堆栈处理函数, 即 recover 方法
type StackTraceHandlerFunc = func(c *fiber.Ctx, e any)
