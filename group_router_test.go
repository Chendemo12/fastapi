package fastapi

import (
	"github.com/Chendemo12/fastapi/pathschema"
	"testing"
	"time"
)

type Clipboard struct {
	Text string `json:"text"`
}

type ExampleGroup struct {
	BaseRouter
}

func (g *ExampleGroup) PathSchema() pathschema.RoutePathSchema { return pathschema.Default() }

func (g *ExampleGroup) GetClipboard(c *Context, day string, time time.Time) (*Clipboard, error) {

	return &Clipboard{}, nil
}

func (g *ExampleGroup) PostClipboard(c *Context, req *Clipboard) (int, error) {

	return 200, nil
}

func TestNewGroupRouteMeta(t *testing.T) {
	meta := NewGroupRouteMeta(&ExampleGroup{})
	err := meta.Init()
	if err != nil {
		panic(err)
	}
}
