package middleware

import (
	"net/http"
	"net/http/httptest"
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
