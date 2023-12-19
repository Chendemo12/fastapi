package fastapi

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Chendemo12/fastapi-tool/logger"
)

// Configuration 配置文件类
type Configuration struct {
	HTTP struct {
		Host string `json:"host" yaml:"host"` // API host
		Port string `json:"port" yaml:"port"` // API port
	}
}

// ServiceContext 全局服务依赖
type ServiceContext struct {
	Conf   *Configuration
	Logger logger.Iface
}

func (c *ServiceContext) Config() any { return c.Conf }

func NewCtx() *ServiceContext {
	conf := &Configuration{}
	conf.HTTP.Host = "0.0.0.0"
	conf.HTTP.Port = "8099"

	return &ServiceContext{Conf: conf, Logger: logger.NewDefaultLogger()}
}

func TestWrapper_DumpPID(t *testing.T) {
	svc := NewCtx()
	svc.Conf.HTTP.Port = "8089"
	app := New(Config{
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})

	go func() {
		time.Sleep(2 * time.Second)

		_bytes, err := os.ReadFile("pid.txt")
		if errors.Is(err, os.ErrNotExist) {
			t.Errorf("dump pid failed: %s", err.Error())
		} else {
			t.Logf("current pid is: %s", string(_bytes))
		}
		app.Shutdown()
	}()

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}

func TestWrapper_Description(t *testing.T) {
	app := New(Config{
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})
	app.SetDescription("一个简单的FastApi应用程序,在启动app之前首先需要创建并替换ServiceContext,最后调用Run来运行程序")

	s := app.Config().Description
	if s == "" {
		t.Errorf("get description failed: %s", s)
	} else {
		t.Logf("get description: %s", s)
	}
}

func TestWrapper_OnEvent(t *testing.T) {
	svc := NewCtx()
	app := New(Config{
		Version:     "1.0.0",
		Description: "",
		Title:       "FastApi Example",
		Debug:       true,
	})

	app.OnEvent("startup", func() { app.Service().Logger().Info("startup event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 1") })
	app.OnEvent("shutdown", func() { app.Service().Logger().Info("shutdown event: 2") })

	app.Run(svc.Conf.HTTP.Host, svc.Conf.HTTP.Port) // 阻塞运行
}
