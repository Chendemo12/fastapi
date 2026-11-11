package test

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Chendemo12/fastapi"
	"github.com/Chendemo12/fastapi/middleware/ginWrapper"
)

// ============================================================================
// 结构体查询参数回归测试
//
// 历史缺陷：结构体字段的校验器名称使用了 Go 字段名（SchemaTitle）而非查询名（JsonName），
// 导致查询参数池取名失败、跳过类型转换，数值/布尔/时间字段必然绑定失败，
// 并进一步触发 ParseJsoniterError 的下标越界 panic。
// ============================================================================

const queryBindPort = "18097"

// QueryBindReq 混合类型的结构体查询参数
type QueryBindReq struct {
	Name    string    `json:"name,omitempty" query:"name"`
	Size    int       `json:"size,omitempty" query:"size"`
	Total   int64     `json:"total,omitempty" query:"total"`
	Weight  float64   `json:"weight,omitempty" query:"weight"`
	Enabled bool      `json:"enabled,omitempty" query:"enabled"`
	StartAt time.Time `json:"startAt,omitempty" query:"startAt"`
}

// QueryBindRouter 混合类型结构体查询参数路由
type QueryBindRouter struct {
	fastapi.BaseGroupRouter
}

func (r *QueryBindRouter) Prefix() string { return "/api/query-bind" }

func (r *QueryBindRouter) GetList(c *fastapi.Context, param *QueryBindReq) (*QueryBindReq, error) {
	resp := *param
	if !resp.StartAt.IsZero() {
		resp.StartAt = time.Unix(resp.StartAt.Unix(), 0).UTC()
	}
	return &resp, nil
}

// StringQueryReq 纯字符串结构体（历史写法，行为必须保持不变）
type StringQueryReq struct {
	Cursor string `json:"cursor,omitempty" query:"cursor"`
	Size   string `json:"size,omitempty" query:"size"`
}

type StringQueryRouter struct {
	fastapi.BaseGroupRouter
}

func (r *StringQueryRouter) Prefix() string { return "/api/string-query" }

func (r *StringQueryRouter) GetList(c *fastapi.Context, param *StringQueryReq) (*StringQueryReq, error) {
	return param, nil
}

func TestStructQueryBind(t *testing.T) {
	app := fastapi.New(fastapi.Config{Title: "query-bind-test", Version: "test"})
	app.SetMux(ginWrapper.Default())
	app.IncludeRouter(&QueryBindRouter{}).IncludeRouter(&StringQueryRouter{})

	go app.Run("127.0.0.1", queryBindPort)
	waitServerReady(t)

	cases := []struct {
		name       string
		url        string
		wantStatus int
		contains   string
	}{
		{"混合类型", "/api/query-bind/list?name=n&size=3&total=9007199254740993&weight=3.5&enabled=true&startAt=2024-01-02", 200, `"total":9007199254740993`},
		{"混合类型-负数值", "/api/query-bind/list?size=-5&weight=-1.25&enabled=false", 200, `"size":-5`},
		{"空参数取零值", "/api/query-bind/list", 200, `{"startAt":"0001-01-01T00:00:00Z"}`},
		{"空字符串参数", "/api/query-bind/list?size=", 200, `{"startAt":"0001-01-01T00:00:00Z"}`},
		{"非法整数", "/api/query-bind/list?size=abc", 422, "size"},
		{"整数溢出", "/api/query-bind/list?size=99999999999999999999", 422, "size"},
		{"非法布尔", "/api/query-bind/list?enabled=notabool", 422, "enabled"},
		{"非法时间", "/api/query-bind/list?startAt=bad", 422, "startAt"},
		{"纯字符串结构体不变", "/api/string-query/list?cursor=c1&size=3", 200, `"size":"3"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := doGet(t, tc.url)
			if status != tc.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", status, tc.wantStatus, body)
			}
			if tc.contains != "" && !contains(body, tc.contains) {
				t.Fatalf("body=%s 未包含 %s", body, tc.contains)
			}
		})
	}
}

func doGet(t *testing.T, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s%s", queryBindPort, path))
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应失败: %v", err)
	}
	return resp.StatusCode, string(data)
}

func waitServerReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/api/string-query/list", queryBindPort))
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("服务启动超时")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
