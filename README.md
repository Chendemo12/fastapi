# FastApi-Golang

- `FastApi`的`Golang`实现;
- 基于`Fiber`;

## Usage:

```bash
go get https://github.com/Chendemo12/fastapi-go
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
# 浏览器打开：http://127.0.0.1:6060/github.com/Chendemo12/fastapi-go
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

# 下载资源文件
#https://fastapi.tiangolo.com/img/favicon.png
#https://cdn.jsdelivr.net/npm/swagger-ui-dist@4/swagger-ui.css
#https://cdn.jsdelivr.net/npm/swagger-ui-dist@4/swagger-ui-bundle.js
#https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js

# 打包资源文件到openapi包
go-bindata -o internal/openapi/css.go --pkg openapi internal/static/...

```

## Examples:

### Guide

- [guide example](example/simple.go)

## 一些常用的API

- 全部`api`可见[`alias.go`](./alias.go)文件；
