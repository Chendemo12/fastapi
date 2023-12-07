package fastapi

import (
	"context"
	"github.com/Chendemo12/fastapi-tool/cronjob"
	"github.com/Chendemo12/fastapi-tool/logger"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/go-playground/validator/v10"
)

// HotSwitchSigint 默认热调试开关
const HotSwitchSigint = 30

const ( // json序列化错误, 关键信息的序号
	jsoniterUnmarshalErrorSeparator = "|" // 序列化错误信息分割符, 定义于 validator/validator_instance.orSeparator
	jsonErrorFieldMsgIndex          = 0   // 错误原因
	jsonErrorFieldNameFormIndex     = 1   // 序列化错误的字段和值
	jsonErrorFormIndex              = 3   // 接收到的数据
)

// EventKind 事件类型
type EventKind string

const (
	StartupEvent  EventKind = "startup"
	ShutdownEvent EventKind = "shutdown"
)

// Event 事件
type Event struct {
	Fc   func()
	Type EventKind // 事件类型：startup 或 shutdown
}

// ------------------------------------------------------------------------------------

// Service Wrapper 全局服务依赖信息
// 此对象由Wrapper启动时自动创建，此对象不应被修改，组合和嵌入，
// 但可通过 setUserSVC 接口设置自定义的上下文信息，并在每一个路由钩子函数中可得
type Service struct {
	logger    logger.Iface        `description:"日志对象"`
	ctx       context.Context     `description:"根Context"`
	validate  *validator.Validate `description:"请求体验证包"`
	openApi   *openapi.OpenApi    `description:"模型文档"`
	cancel    context.CancelFunc  `description:"取消函数"`
	scheduler *cronjob.Scheduler  `description:"定时任务"`
	addr      string              `description:"绑定地址"`
}

// 替换日志句柄
//
//	@param	logger	logger.Iface	日志句柄
func (s *Service) setLogger(logger logger.Iface) *Service {
	s.logger = logger
	return s
}

// Addr 绑定地址
//
//	@return	string 绑定地址
func (s *Service) Addr() string { return s.addr }

// RootCtx 根 context
//
//	@return	context.Context 整个服务的根 context
func (s *Service) RootCtx() context.Context { return s.ctx }

// Logger 获取日志句柄
func (s *Service) Logger() logger.Iface { return s.logger }

// Done 监听程序是否退出或正在关闭，仅当程序关闭时解除阻塞
func (s *Service) Done() <-chan struct{} { return s.ctx.Done() }

// Scheduler 获取内部调度器
func (s *Service) Scheduler() *cronjob.Scheduler { return s.scheduler }

// Validate 结构体验证
//
//	@param	stc	any	需要校验的结构体
//	@param	ctx	any	当校验不通过时需要返回给客户端的附加信息，仅第一个有效
//	@return
func (s *Service) Validate(stc any, ctx ...map[string]any) *Response {
	err := s.validate.StructCtx(s.ctx, stc)
	if err != nil { // 模型验证错误
		err, _ := err.(validator.ValidationErrors) // validator的校验错误信息

		if nums := len(err); nums == 0 {
			return nil // TODO: error
		} else {
			ves := make([]*openapi.ValidationError, nums) // 自定义的错误信息
			for i := 0; i < nums; i++ {
				ves[i] = &openapi.ValidationError{
					Loc:  []string{"body", err[i].Field()},
					Msg:  err[i].Error(),
					Type: err[i].Type().String(),
				}
				if len(ctx) > 0 {
					ves[i].Ctx = ctx[0]
				}
			}
			return nil // TODO: error
		}
	}

	return nil
}
