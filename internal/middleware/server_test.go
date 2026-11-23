package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	readDeadline  time.Time
	writeDeadline time.Time
}

func (r *deadlineRecorder) SetReadDeadline(deadline time.Time) error {
	r.readDeadline = deadline
	return nil
}

func (r *deadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	r.writeDeadline = deadline
	return nil
}

func TestLoggerSkipsHealthChecks(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger())
	router.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.HEAD("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.HEAD("/api/v1/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/other", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/healthz", nil),
		httptest.NewRequest(http.MethodHead, "/healthz", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/health", nil),
		httptest.NewRequest(http.MethodHead, "/api/v1/health", nil),
		httptest.NewRequest(http.MethodGet, "/other", nil),
	} {
		router.ServeHTTP(httptest.NewRecorder(), request)
	}
	if strings.Contains(logs.String(), "/healthz") || strings.Contains(logs.String(), "/api/v1/health") {
		t.Fatalf("健康检查不应产生访问日志: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "/other") {
		t.Fatalf("其他请求访问日志应保留: %s", logs.String())
	}
}

func TestRecoveryLogsStackWithoutExposingIt(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) { panic("sensitive panic") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("panic 响应状态错误: %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "sensitive panic") || strings.Contains(response.Body.String(), "goroutine") {
		t.Fatalf("panic 详情不应出现在响应中: %s", response.Body.String())
	}
	if !strings.Contains(logs.String(), "sensitive panic") || !strings.Contains(logs.String(), "goroutine") {
		t.Fatalf("服务端日志应包含 panic 与堆栈: %s", logs.String())
	}
}

func TestRequestDeadlineUsesUnderlyingHTTPConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestDeadline(time.Minute))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.readDeadline.IsZero() || recorder.writeDeadline.IsZero() {
		t.Fatal("Gin ResponseWriter 未将 deadline 传递到底层 HTTP 连接")
	}
}

func TestBodyLimitSkipsGETAndRejectsLargeContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BodyLimit(10))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.POST("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.ContentLength = 100
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("GET 不应套请求体限制: %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/", nil)
	request.ContentLength = 100
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("超限 Content-Length 应返回 413: %d", response.Code)
	}
}
