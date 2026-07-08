# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Lint

```bash
go build ./...          # build all packages
go test ./...           # run all tests
go test -v ./...        # run all tests with verbose output
go test -v ./openapi/   # run tests for a specific package
go vet ./...            # static analysis
```

## Architecture

This is **Go-FastApi** (`github.com/Chendemo12/fastapi`), a Go HTTP framework inspired by Python's FastAPI. It is a **wrapper/adapter layer**, not a standalone HTTP server — it sits on top of Fiber or Gin and provides declarative routing, automatic OpenAPI 3.1 doc generation, and automatic request parameter validation.

### Core concepts

- **`Wrapper`** (alias `FastApi`) in `app.go` is the application root. It owns configuration, registered routes, dependency hooks, and the OpenAPI generator. Lifecycle: `New()` → `SetMux()` → `IncludeRouter()` → `Run()`.

- **`MuxWrapper` / `MuxContext`** (in `mux.go`) are the HTTP engine abstraction interfaces. Adapters exist for Fiber (`middleware/fiberWrapper/`) and Gin (`middleware/ginWrapper/`). `MuxWrapper` handles server-level concerns (Listen, Shutdown, BindRoute); `MuxContext` handles per-request concerns (Query, Params, ShouldBind, JSON, SSE, File).

- **`GroupRouter`** (in `group_router.go`) is the declarative route interface. Users embed `BaseGroupRouter` and define methods named with HTTP verb prefixes (e.g., `GetAppTitle` → `GET /app-title`). Routes are discovered via reflection at startup. Method naming → path translation is controlled by `PathSchema()` (default: `LowerCaseDash`).

- **Request lifecycle** (`hook.go`, `Wrapper.Handler`): route lookup → acquire ctx from pool → pre-dependency hooks → validation pipeline (path params → query params → struct query → body) → post-dependency hooks → route handler → response validation → before-write hook → serialize & write response → release ctx.

- **`ModelBinder`** (in `validate.go`) handles type coercion and validation for every parameter kind. `RouteParamType()` maps to OpenAPI types. Implementations: `IntModelBinder`, `FloatModelBinder`, `BoolModelBinder`, `JsonModelBinder` (uses `go-playground/validator`), `FileModelBinder`, `TimeModelBinder`, etc.

- **OpenAPI generation** (in `openapi/`): `OpenApi` struct builds a full OpenAPI 3.1.0 document from registered routes. `RouteSwagger` captures per-route metadata. `BaseModelMeta` / `BaseModelField` use reflection to build JSON schemas from Go structs. Served at `/openapi.json`, with Swagger UI at `/docs` and ReDoc at `/redoc`.

- **`sync.Pool`** is used pervasively for `Context`, `Response`, and mux-specific context wrappers to minimize GC pressure.

### Key patterns

- All handler methods return `(any, error)`. A non-nil error triggers `RouteErrorFormatter`; a nil error causes the return value to be serialized as JSON (or text/SSE/file depending on type).
- `Wrapper` methods return `*Wrapper` for fluent chaining (e.g., `app.SetMux(m).IncludeRouter(r).UsePrevious(h)`).
- The `validate` struct tag is used (not `validate` spelled differently) for `go-playground/validator` integration.
- Package-level function variables (`Info`, `Debug`, `Error` in `logger.go`) can be replaced via `ReplaceLogger()`.
- Route path formatting is pluggable via `pathschema.RoutePathSchema` interface (`LowerCaseDash`, `Backslash`, `LowerCase`, `Raw`).

### File guide

| File/Dir | Role |
|---|---|
| `app.go` | `Wrapper` struct, `Config`, `New()`, `Run()`, shutdown |
| `hook.go` | Request lifecycle: validation pipeline, response writing |
| `group_router.go` | `GroupRouter` interface, `BaseGroupRouter`, route scanning |
| `validate.go` | All `ModelBinder` implementations |
| `router.go` | `RouteIface`, `ScanHelper`, binder inference from struct tags |
| `mux.go` | `MuxWrapper` and `MuxContext` interfaces |
| `ctx.go` | Per-request `Context` struct |
| `finder.go` | Route lookup (hash-based and simple implementations) |
| `swagger.go` | Swagger UI / ReDoc static route setup |
| `response.go` | `Response` struct with sync.Pool |
| `file.go` | File upload/download types |
| `sse.go` | SSE message type |
| `consts.go` | Type aliases (`H`, `Dict`, `Ctx`, `BaseRouter`) |
| `openapi/` | OpenAPI 3.1 document generation subpackage |
| `pathschema/` | Pluggable URL path formatting strategies |
| `middleware/fiberWrapper/` | Fiber HTTP engine adapter |
| `middleware/ginWrapper/` | Gin HTTP engine adapter |
| `utils/` | JSON (sonic), string, reflection, generics helpers |
| `test/` | Integration tests with Fiber and Gin |
