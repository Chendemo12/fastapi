# CHANGELOG

## 0.3.1 - (2025-08-15)

### Feat

- 支持上传文件；
- 支付下载文件；

### BREAKING

- 移除`Wrapper.ActivateHotSwitch`相关方法。
- 移除`Wrapper.RootCtx`方法。
- 移除`Wrapper.SetRouteErrorStatusCode`方法。
- 移除`Wrapper.SetRouteErrorResponse`方法。
- **不再支持非结构体参数**，请使用结构体来定义查询参数或请求体参数。

## 0.3.0 - (2025-05-04)

### Feat

- 升级go版本`v1.23`

## 0.2.12 - (2024-12-17)

### Fix

- 修复默认的错误处理handle`defaultRouteErrorFormatter`覆盖`Context.Status设置状态码后被重置为500`的错误；

## 0.2.11 - (2024-07-10)

### Feat

- `fiberWrapper.Default` 新增参数；
- 更新 `fiber:2.52.5`；

## 0.2.10 - (2024-05-02)

### Fix

- 修改`SetRouteErrorFormatter`不生效的错误；
- `Context.Status`方法可以正常起作用；

## 0.2.9 - (2024-05-02)

### Feat

- 修改默认logger
- 新增`MuxContext.GetHeader`
- `FiberContext`实现`MuxContext`全部方法
- 初步实现`GinMux`

### BREAKING

- (2024-05-02) 移除`Context.Logger`方法;

## 0.2.8 - (2024-04-22)

### Feat

- expose MuxContext;
- 不再打算支持泛型路由;
- ~~(2024-04-26) 路由组新增`ErrorFormatter`方法~~;
- (2024-04-26) 支持设置全局错误处理函数`RouteErrorFormatter`及其数据模型，并为其生成文档;

### Fix

- (2024-04-27) 依赖函数返回错误时同样通过`RouteErrorFormatter`进行格式化

### BREAKING

- (2024-04-26) 移除`Wrapper.Debug`参数;

## 0.2.7 - (2024-04-19)

### Feat

- (2024-04-19) 新增方法`fiber.NewAuthInterceptor`

### Fix

- (2024-04-19) 修复`Wrapper.Shutdown`的错误；

## 0.2.6 - (2024-03-26)

### Feat

- 增加报错信息以显式提醒`POST/PATCH/PUT`方法缺少必要参数；
- (2024-04-17) 修改logger, 废弃Context.Logger()方法, 但是可以通过`fastapi.Info`获得;
- (2024-04-19) 新增`路由错误处理函数`;
- (2024-04-19) 新增`MIME`类型;
- (2024-04-19) 修改`Wrapper.write` 和 `Context.Response`;
- (2024-04-19) 新增方法 `Wrapper.SetHotListener`;

## 0.2.5 - (2024-03-24)

### Feat

- upgrade version;

## 0.2.4 - (2024-03-22)

### Fix

- 修改UseXXX系列方法的解释说明;

### Feat

- update to fiver:v2.52.2;
- update to validator:v10.19.0;

## 0.2.3 - (2024-03-20)

### Fix

- 修复请求context自动派生功能异常的错误;
- 可以通过Context获得wrapper根context;

## 0.2.2 - (2024-01-25)

### Feat

- Context 新增Set,Get替代实现, 用于在MuxContext未实现此方法时调用;
- Wrapper 新增写流前钩子方法;

## 0.2.1 - (2024-01-24)

### Fix

- 路由组路由函数存在错误时，仅在未手动设置响应状态码的情况下才设置默认值；

### Feat

- 结构体查询参数支持嵌入结构体；
- 数据模型支持嵌入结构体；

## 0.2.0 - (2023-11-26 ~ 2024-01-19)

### BREAKING

- 项目重构;
- 删除原路由注册方法;
- 新增`结构体路由组`式注册方法;
- 修改原路由注册方式为泛型模式;
- 修改校验流程;
- 支持结构体参数文档生成和数据校验;
- 支持json请求体的创建和数据校验;
- 支持对json响应体的参数校验;
- 移除`scheduler`;
- 支持time.Time类型的文档生成和查询参数校验;
- 对于路由组路由来说，允许通过方法重载查询参数名称;
- (2024-01-10) 支持泛型结构体的文档生成，但是不支持泛型结构体嵌套泛型结构体的文档的生成;
- (2024-01-19) 支持带数字的路由模式；

## 0.1.8 - (2023-10-20)

### Feat

- 增加默认的CORS配置;

## 0.1.7 - (2023-10-18)

### Feat

- 升级`fiber:v2.50.0`

### Fix

- 修复当字段为数组基本类型时swagger文档的错误;

## 0.1.6 - (2023-09-15)

### Feat

- 升级`fastapi-tool:v0.1.1`

## 0.1.5 - (2023-09-12)

### Fix

- 修复匿名结构体文档生成的错误;
- 删除openapi的部分接口;
- 合并godantic到openapi;

## 0.1.4 - (2023-09-05)

### Feat

- 新增router系列接口, 简化路由注册接口;
- 更新openapi到v3.1.0;
- 更新FastAPI文档资源到V5版本;

### TODO:

- redoc 页面出现问题，暂无法显示

## 0.1.3 - (2023-07-26)

### Refactor

- remove `types`, `example`

### Fix

- 修复获取未命名的结构体的模型文档的错误，对于未命名的结构体类型，为其分配一个虚假的结构体名称；

## 0.1.2 - (2023-06-28)

### Refactor

- 移除并引入`fastapi-tool`;

## 0.1.1 - (2023-03-30)

### Refactor

- 固化模型校验方法以减少运行时判断，从而提高性能;
- 修改`Route`接口，增加`Options`可选参数;

## 0.1.0 - (2023-03-28)