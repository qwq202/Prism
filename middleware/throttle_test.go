package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func TestThrottleMiddlewareAllowsRequestWhenRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cache := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		MaxRetries:   -1,
		DialTimeout:  time.Millisecond,
		ReadTimeout:  time.Millisecond,
		WriteTimeout: time.Millisecond,
	})
	t.Cleanup(func() {
		_ = cache.Close()
	})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("cache", cache)
		c.Next()
	})
	router.Use(ThrottleMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": true})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.RemoteAddr = "203.0.113.10:4321"

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected request to pass through, got status %d body %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != `{"status":true}` {
		t.Fatalf("expected handler response, got %s", recorder.Body.String())
	}
}
