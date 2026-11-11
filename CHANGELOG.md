# CHANGELOG

## 0.3.8 - (2026-11-11)

### Fix

- **修复结构体查询参数（GET/DELETE）数值、布尔、时间类型必然失败的历史缺陷**：
    - `ScanHelper.InferQueryBinder` 对结构体字段（`QModel.InStruct`）改用查询名（`JsonName`）作为校验器名称，
      此前使用 Go 字段名（`SchemaTitle`），导致 `Context.queryFields` 取名失败、**类型转换被整段跳过**，
      参数始终以 URL 原始字符串进入结构体绑定；
    - 典型表现：`?size=3&unreadOnly=true` 返回 500（并触发 panic）而非 200；`?size=abc` 返回 500 而非 422；
      `?startAt=2024-01-02` 等非 RFC3339 的 `time.Time` 参数同样失败；
    - 新增 `ScanHelper.InferParamBinderWithName`，允许显式指定校验器名称；`InferParamBinder` 签名与行为保持不变；
    - 附带修正：结构体字段校验错误的 `loc` 由 Go 字段名变为查询名，与 OpenAPI 文档声明一致；
- **修复 `ParseJsoniterError` 下标越界 panic**：
    - 分割后的长度判断由恒真的 `len(msgs) >= jsonErrorFieldMsgIndex+1`（即 `>=1`）改为 `len(msgs) > jsonErrorFormIndex`；
    - 该解析逻辑仅适用于 json-iterator 的错误文本，本项目实际使用 sonic / encoding/json（错误文本不含 `|`），
      修复前任何非 jsoniter 文本都会越界 panic；
    - 解析失败时以原始错误文本兜底，`Msg` 不再为空；sonic 的多行上下文只保留首行；
    - 移除 `msgs[0][len(ve.Loc[1])+2:]` 的潜在越界写法；
    - 该函数同时服务请求体校验（fiber 适配器）与 multipart json 参数校验，一并修复；
- **修复校验错误上下文复用包级 map 导致的跨请求污染与 data race**：
    - `whereClientError` / `whereServerError` 不再直接作为 `ValidationError.Ctx` 暴露给调用方；
    - 新增 `clientErrorCtx()` / `serverErrorCtx()`，每次返回全新 map；`ParseValidatorError` 内部同样改为局部新建；
    - 此前一次请求体校验错误会把 `param description` 写入全局 map，导致后续**查询参数**错误的响应里出现残留字段；
- **修复 `hook.go` 中的裸类型断言 panic**：`queryParamsValidate` 中 `v.(string)` 改为带 `ok` 判断，
  避免结构体字段与函数参数同名时，第二个校验器收到已转换的 `int64`/`bool` 而 panic；
- 修复 `go vet` 告警：`GroupRouterMeta.scanMethod` 中 `swagger.Tags = append(r.tags)` 改为追加语义，
  避免多条路由共享同一个 tags 切片。

### Test

- 新增 `validate_error_test.go`：覆盖 `ParseJsoniterError` 对纯文本 / encoding/json / sonic 多行 /
  `|` 分隔不足 / jsoniter 格式文本的处理（均不得 panic），以及 `Ctx` 在多次调用间相互隔离；
- 新增 `test/query_bind_test.go`：以真实 HTTP 覆盖结构体查询参数的
  `string/int/int64/float64/bool/time.Time` 绑定（含负数、`int64` 精度、空值与缺参取零值），
  以及非法整数 / 整数溢出 / 非法布尔 / 非法时间返回 422 且 `loc` 为查询名，
  同时回归"纯字符串结构体"的历史写法行为不变。

## 0.3.7 - (2026-11-09)

### BREAKING

- MCP `AuthFunc` 认证失败时不再固定返回 HTTP 200：
    - 未调用 `c.Status()` 时，默认返回 **401 Unauthorized**；
    - 可通过 `c.Status(code)` 覆盖，例如凭证有效但 scope 不足时返回 403；
    - 可通过 `c.Status(http.StatusOK)` 显式保留旧的 200 行为；
- MCP 批量请求（JSON-RPC batch）中任一消息认证失败时，整个 HTTP 请求直接返回认证失败状态码，
  不再继续处理并返回 200 + 逐条 JSON-RPC 错误（fail-fast）；

### Feat

- MCP `AuthFunc` 支持自定义认证失败状态码：`authenticate` 读取 `Context.Response().StatusCode`
  并透传至 `writeResponse`；非法状态码（小于 100 或大于 599）回退为 401；
- MCP `AuthFunc` 可通过 `c.MuxContext().Header(key, value)` 设置 `WWW-Authenticate` 等响应头，
  框架原样透传，不自动生成 challenge（避免 API Key 模式被误认为 OAuth）；
- `MCPHandler.writeResponse` 增加可选 HTTP 状态码参数，未传入时保持原有 200 行为；
- `MCPHandler.authenticate` 返回值增加 HTTP 状态码，供 `serveOne` / `serveBatch` 统一写出；

### Docs

- 补充 `MCPProvider.AuthFunc` 文档：说明认证失败默认状态码、`c.Status` 覆盖方式及响应头设置方式；
- 补充 `Context.Status` 文档：说明其在 MCP `AuthFunc` 中用于指定认证失败状态码；

### Test

- 新增 MCP 认证失败测试：
    - 未设置状态码时默认返回 401；
    - `c.Status(403)` 生效；
    - `WWW-Authenticate` 响应头原样保留；
    - 批量请求认证失败时整体拒绝，不返回逐条错误；
    - `Response` 池复用不会把上一次认证失败的状态码带入下一次请求；

## 0.3.6 - (2026-10-24)

### BREAKING

- MCP 传输层改为**无状态**：不再签发或校验 `mcp-session-id`，移除 10 分钟固定会话超时；每个请求独立鉴权，客户端无需先 `initialize` 即可调用工具；
- MCP 协议版本由 `2024-11-05` 升级为 `2025-03-26`（Streamable HTTP 正式版本），并在 `initialize` 中按客户端请求版本协商；

### Feat

- MCP 支持 JSON-RPC 批量请求（数组），通知类消息（`notifications/*`）按规范返回 202 且无响应体；`initialize` 不允许出现在批量请求中；
- MCP 新增 `ping` 方法，返回空结果对象；
- MCP 端点新增 `GET /mcp`（无 SSE，返回 405）与 `DELETE /mcp`（无会话，返回 405），符合 Streamable HTTP 对同一端点的方法要求；

### Fix

- 修复 MCP `prompts/get` 返回的 `messages` 结构不符合规范的问题（补齐 `role` 与嵌套 `content`）；
- 修复 `resources/read` 返回结构与错误码：补充 `mimeType`，移除多余的 `type` 字段，资源不存在改为 `-32002`；
- 修复解析错误、非法请求仍返回 HTTP 200 的问题，现分别返回 400；
- 未知工具错误信息统一为 `Unknown tool: <name>`；
- 校验 JSON-RPC 请求 `id` 类型（仅允许字符串/数字/null，非法时返回 `-32600`），批量请求同样适用；
- 同名 Tool/Resource/Prompt 冲突时仅保留首个并打印告警，避免静默覆盖；

## 0.3.5-beta5 - (2026-10-24)

### Fix

- 修复 MCP Tool `inputSchema` 生成时把 `json:"field,omitempty"` 的逗号选项一并当作属性名，导致属性名形如 `field,omitempty` 的问题；
- 修复 MCP Tool `inputSchema` 缺少 `[]string`/`[]int`/`[]struct` 数组字段的问题（原 `StructToQModels` 会跳过切片/数组类型）；
- `MCPToolMeta.Call` 在调用 Tool 方法前接入结构体入参校验，新增 `SetMCPValidateLang("zh"|"en")` 以配置中英文校验错误提示（默认英文）；

## 0.3.5-beta4 - (2026-07-30)


### Fix

- 修复具名非结构体类型（如 `type JsonData []*BenefitItem`）作为结构体字段时，OpenAPI 元数据扫描器
  将解引用后的切片类型误当结构体处理（`reflect.NumField()`） 导致 panic 的问题；

## 0.3.5-beta3 - (2026-07-14)

### Fix

- 修复`v0.3.5:r.RouteParamType == RouteParamRequest && r.Pkg == FileRequestPkg`引发的文件上传的错误；

## 0.3.5-beta2 - (2026-07-13)

### BREAKING

- rename `FastApiContext` to `WrapperContext`

## 0.3.5-beta1 - (2026-07-12)

### Feat

- **MCP (Model Context Protocol) 支持**：内置 MCP 能力，采用 Streamable HTTP 传输协议，单端点 `POST /mcp`；
    - 通过结构体反射方式定义 MCP Provider，嵌入 `BaseMCPProvider` 并遵循 `Tool*`/`Resource*`/`Prompt*` 方法名前缀即可自动发现；
    - 自实现 JSON-RPC 2.0 协议层，无外部 SDK 依赖，通过反射直接调用 Go 方法；
    - 支持 Tool（工具调用）、Resource（资源读取）、Prompt（提示模板）三种 MCP 原语；
    - `MCPProvider` 接口支持 `Prefix`、`Tags`、`PathSchema`、`Summary`、`Description`、`ResourceURI`、`AuthFunc` 七个扩展点；
    - 默认命名规则采用 `pathschema.LowerCaseUnderline`（snake_case），可通过重写 `PathSchema()` 自定义；
    - `AuthFunc(c *Context) error` 支持按 Provider 粒度的认证，返回非 nil error 即拒绝请求；
    - MCP 与 HTTP 路由（GroupRouter）可并存，互不干扰；
    - 新增完整示例 `test/mcp_server_example.go`（含公开和认证两类 Provider）；
    - 新增文档 `docs/mcp.md`，RESTful API 文档独立为 `docs/restful-api.md`；

### BREAKING

- **移除 `Config.ContextAutomaticDerivationDisabled` 配置项**：改为直接使用底层 HTTP 引擎（Fiber/Gin）的请求级
  `context.Context`，不再额外创建 `context.WithCancel`，`c.Context()` 和 `c.Done()` 公开 API 不变；
- **`MuxContext` 接口新增 `RequestContext() context.Context` 方法**：自定义 `MuxContext` 实现者需添加此方法，当前框架已为
  Fiber、Gin 适配器完成实现；

## 0.3.4 - (2026-07-09)

### Fix

- 修复 `StringsToFloats` 因错误预分配导致返回结果包含前导零值的问题;
- 修复 SSE 推流时 goroutine 在 FiberContext 回收后访问空指针导致 panic 的问题;
- 修复递归结构体 (如 `type Node struct { Next *Node }`) 作为 API 模型时栈溢出的问题;
- 修复 `ParseJsoniterError` 对 sonic 错误消息格式的脆弱索引，增加 bounds check;
- 修复 Swagger/Redoc UI 全局模板变量多协程并发写导致的 data race;
- `releaseCtx` 补充遗漏的 `queryStruct = nil` 清理;

### Perf

- `fastapi.Context` 结构体重排：删死代码 `appCtx`，`sseOnce` 指针改值类型，热点字段前置，体积缩小 8% (144→132 bytes);
- `acquireCtx`/`releaseCtx` 优化：locker 预分配复用、pathFields/queryFields map 清空替换 nil 重置，每请求省 3 次堆分配;
- `DateTimeModelBinder`/`TimeModelBinder`/`DateModelBinder` 的 layout 切片提升为包级变量，省每请求切片分配;
- `ParseJsoniterError`/`ParseValidatorError` 移除死 map 分配;
- `DefaultLogger.Errorf`/`Warnf`/`Debugf` 改为 `fmt.Sprintf` 替代 `fmt.Errorf`，省 error 包装分配;
- `IsGenericModel` 正则编译从函数内提升为包级变量，消除启动期重复编译;
- `SplitWords` 内联空串过滤，省 `SliceFilter` 二次切片分配;
- `scanObjectSwagger` 移除死 map 分配;
- Fiber/Gin 的 `BindRoute` 5 个相同闭包提取为 1 个;
- `FiberContext.GetHeader` 用 `c.ctx.Get(key)` 替代 `GetReqHeaders()[key]`，省全量 header map 分配;

### Refactor

- `fastapi.Context` 与 `MuxContext` 所有权反转：`FiberContext`/`GinContext` 嵌入 `fastapi.Context`，消除双 pool，每请求
  pool 操作从 4 次降为 2 次;
- `MuxContext` 接口新增 `WrapperContext() *Context` 方法，由适配器返回嵌入的 Context;
- `fastapi.Context` 新增 `NewContext()`/`InitContext()`/`ResetContext()`，由中间件 pool 管理生命周期;
- `Wrapper` 移除 Context pool，委托给各 mux 适配器;
- `Finder.Get` 方法签名从 `Get(id string)` 改为 `Get(method, path string)`，内部改用 `map[routeKey]T` 实现 O(1) 精确匹配;
- `openapi/metadata.go` `BaseModelMeta` 新增 `visited` 字段用于循环引用检测;

## 0.3.3 - (2025-08-31)

### Feat

- 初步支持`SSE`推流，需底层`mux`支持；
- 切换json序列化为`sonic`，因此不支持`go:v1.24.0`，详情见sonic文档;

## 0.3.2 - (2025-08-20)

### Fix

- 修复设置`DisableResponseValidate`不生效的错误；

## 0.3.1 - (2025-08-17)

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