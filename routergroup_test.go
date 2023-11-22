package fastapi

import "testing"

type ExampleGroup struct {
	BaseRouter
}

func (g *ExampleGroup) PathSchema() RoutePathSchema { return DoNothing{} }

func (g *ExampleGroup) GetClipboard(c *Context, day string) (*Clipboard, error) {

	return &Clipboard{}, nil
}

func TestIncludeRouter(t *testing.T) {
	IncludeRouter(&ExampleGroup{})
}
