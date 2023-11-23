package fastapi

import "testing"

type ExampleGroup struct {
	BaseRouter
}

func (g *ExampleGroup) PathSchema() RoutePathSchema { return DoNothing{} }

func (g *ExampleGroup) GetClipboard(c *Context, day string) (*Clipboard, error) {

	return &Clipboard{}, nil
}

func TestScanRouterMethod(t *testing.T) {
	meta := ScanGroupRouter(&ExampleGroup{})
	println(len(meta.routes))
	println(meta.routes[0].route.swagger.Url)
	println(meta.routes[0].route.swagger.Method)
	println(meta.routes[0].route.swagger.Tags[0])
	ScanGroupRoute(meta)
}
