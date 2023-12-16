package fastapi

import (
	"context"
	"github.com/Chendemo12/fastapi-tool/cronjob"
	"github.com/Chendemo12/fastapi-tool/logger"
	"github.com/Chendemo12/fastapi/openapi"
	"github.com/go-playground/validator/v10"
	jsoniter "github.com/json-iterator/go"
)

var dLog logger.Iface = logger.NewDefaultLogger()

func Logger() logger.Iface { return dLog }

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
	ctx       context.Context    `description:"根Context"`
	openApi   *openapi.OpenApi   `description:"模型文档"`
	cancel    context.CancelFunc `description:"取消函数"`
	scheduler *cronjob.Scheduler `description:"定时任务"`
	addr      string             `description:"绑定地址"`
}

// 替换日志句柄
//
//	@param	logger	logger.Iface	日志句柄
func (s *Service) setLogger(logger logger.Iface) *Service {
	dLog = logger
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
func (s *Service) Logger() logger.Iface { return dLog }

// Done 监听程序是否退出或正在关闭，仅当程序关闭时解除阻塞
func (s *Service) Done() <-chan struct{} { return s.ctx.Done() }

// Scheduler 获取内部调度器
func (s *Service) Scheduler() *cronjob.Scheduler { return s.scheduler }

func LazyInit() {
	// 初始化默认结构体验证器
	defaultValidator = validator.New()
	defaultValidator.SetTagName(openapi.ValidateTagName)

	// 初始化结构体查询参数方法
	var queryStructJsonConf = jsoniter.Config{
		IndentionStep:                 0,                    // 指定格式化序列化输出时的空格缩进数量
		EscapeHTML:                    false,                // 转义输出HTML
		MarshalFloatWith6Digits:       true,                 // 指定浮点数序列化输出时最多保留6位小数
		ObjectFieldMustBeSimpleString: true,                 // 开启该选项后，反序列化过程中不会对你的json串中对象的字段字符串可能包含的转义进行处理，因此你应该保证你的待解析json串中对象的字段应该是简单的字符串(不包含转义)
		SortMapKeys:                   false,                // 指定map类型序列化输出时按照其key排序
		UseNumber:                     false,                // 指定反序列化时将数字(整数、浮点数)解析成json.Number类型
		DisallowUnknownFields:         false,                // 当开启该选项时，反序列化过程如果解析到未知字段，即在结构体的schema定义中找不到的字段时，不会跳过然后继续解析，而会返回错误
		TagKey:                        openapi.QueryTagName, // 指定tag字符串，默认情况为"json"
		OnlyTaggedField:               false,                // 当开启该选项时，只有带上tag的结构体字段才会被序列化输出
		ValidateJsonRawMessage:        false,                // json.RawMessage类型的字段在序列化时会原封不动地进行输出。开启这个选项后，json-iterator会校验这种类型的字段包含的是否一个合法的json串，如果合法，原样输出；否则会输出"null"
		CaseSensitive:                 false,                // 开启该选项后，你的待解析json串中的对象的字段必须与你的schema定义的字段大小写严格一致
	}
	structQueryBind = &StructQueryBind{json: queryStructJsonConf.Froze()}
}
