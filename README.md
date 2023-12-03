# FastApi-Golang

- `FastApi`的`Golang`实现;
- 基于`Fiber`;

## Usage:

```bash
go get https://github.com/Chendemo12/fastapi
```

### 查看在线文档

```bash
# 安装godoc
go install golang.org/x/tools/cmd/godoc@latest
godoc -http=:6060

# 或：pkgsite 推荐
go install golang.org/x/pkgsite/cmd/pkgsite@latest
cd fastapi-go/
pkgsite -http=:6060 -list=false
# 浏览器打开：http://127.0.0.1:6060/github.com/Chendemo12/fastapi
```

### `struct`内存对齐

```bash
go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest

fieldalignment -fix ./... 
```

### 打包静态资源文件

```shell
# 安装工具
go get -u github.com/go-bindata/go-bindata/...
go install github.com/go-bindata/go-bindata/...

# 下载资源文件
#https://fastapi.tiangolo.com/img/favicon.png
#https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css
#https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js
#https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js

# 打包资源文件到openapi包
go-bindata -o openapi/css.go --pkg openapi internal/static/...

```

## Guide:

### 面向对象式路由：GroupRouter

```go
package main

type QueryParamRouter struct {
	BaseRouter
}

func (r *QueryParamRouter) Prefix() string { return "/api/query-param" }

func (r *QueryParamRouter) IntQueryParamGet(c *Context, age int) (int, error) {
	return age, nil
}

```

1. 对于请求参数

- [x] 识别函数参数作为查询参数, (GET/DELETE/POST/PUT/PATCH)。
  > 由于反射限制，无法识别函数参数名称，因此显示在文档上的参数名为随机分配的，推荐通过结构体实现。
- [x] 将最后一个结构体解析为查询参数, (GET/DELETE), 推荐。
    ```go
    package main
    
    type LogonQuery struct {
        Father string `query:"father" validate:"required" description:"姓氏"` // 必选查询参数
        Name   string `query:"name" description:"姓名"`                       // 可选查询参数
    }
    ```
- [x] 将最后一个结构体解析为请求体, (POST/PUT/PATCH)。
- [x] 将除最后一个参数解析为查询参数（包含结构体查询参数）, (POST/PUT/PATCH)。
- [x] 允许重载API路由以支持路径参数, (GET/DELETE/POST/PUT/PATCH)。

- 在任何时候，通过添加方法`SchemaDesc() string`以为模型添加文本说明

## 一些常用的API

- 全部`api`可见[`alias.go`](./alias.go)文件；
