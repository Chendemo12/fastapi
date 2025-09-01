package test

import (
	"testing"

	"github.com/Chendemo12/fastapi/middleware/fiberWrapper"
)

func TestGinMux(t *testing.T) {
	mux := fiberWrapper.Default()
	app := CreateApp(mux)
	app.Run("0.0.0.0", "8090") // 阻塞运行
}
