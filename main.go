package main

import (
	"chat/adapter"
	"chat/addition"
	"chat/admin"
	"chat/auth"
	"chat/billing"
	"chat/channel"
	"chat/cli"
	"chat/connection"
	"chat/globals"
	"chat/manager"
	"chat/manager/conversation"
	"chat/manager/memory"
	"chat/middleware"
	"chat/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

const healthCheckTimeout = 2 * time.Second

type healthDependency struct {
	Status string `json:"status"`
}

func normalizeAllowedOriginHost(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}

	if parsed, err := url.Parse(origin); err == nil && parsed.Host != "" {
		origin = parsed.Host
	} else if parsed, err := url.Parse("//" + origin); err == nil && parsed.Host != "" {
		origin = parsed.Host
	} else {
		return ""
	}

	origin = strings.TrimSuffix(origin, "/")
	if strings.HasPrefix(origin, "www.") {
		origin = origin[4:]
	}
	return origin
}

func readCorsOrigins() {
	origins := viper.GetStringSlice("allow_origins")
	if len(origins) > 0 {
		globals.AllowedOrigins = make([]string, 0, len(origins))
		for _, origin := range origins {
			if host := normalizeAllowedOriginHost(origin); host != "" {
				globals.AllowedOrigins = append(globals.AllowedOrigins, host)
			}
		}
	}
}

func checkDatabase(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.PingContext(ctx)
}

func checkRedis(ctx context.Context, cache *redis.Client) error {
	if cache == nil {
		return errors.New("redis is not initialized")
	}
	return cache.Ping(ctx).Err()
}

func dependencyHealth(ctx context.Context, check func(context.Context) error) healthDependency {
	if err := check(ctx); err != nil {
		return healthDependency{Status: "unavailable"}
	}

	return healthDependency{Status: "ok"}
}

func healthHTTPStatus(dependencies ...healthDependency) int {
	for _, dependency := range dependencies {
		if dependency.Status != "ok" {
			return http.StatusServiceUnavailable
		}
	}
	return http.StatusOK
}

func buildHealthResponse(ctx context.Context, db *sql.DB, cache *redis.Client) (int, gin.H) {
	databaseHealth := dependencyHealth(ctx, func(ctx context.Context) error {
		return checkDatabase(ctx, db)
	})
	redisHealth := dependencyHealth(ctx, func(ctx context.Context) error {
		return checkRedis(ctx, cache)
	})

	status := healthHTTPStatus(databaseHealth, redisHealth)
	overall := "ok"
	if status != http.StatusOK {
		overall = "unavailable"
	}

	return status, gin.H{
		"status":   overall,
		"database": databaseHealth,
		"redis":    redisHealth,
	}
}

func healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), healthCheckTimeout)
		defer cancel()

		status, response := buildHealthResponse(ctx, connection.DB, connection.Cache)
		c.JSON(status, response)
	}
}

func registerHealthRoutes(engine *gin.Engine) {
	handler := healthHandler()
	engine.GET("/health", handler)
	if viper.GetBool("serve_static") {
		engine.GET("/api/health", handler)
	}
}

func registerApiRouter(engine *gin.Engine) {
	var app *gin.RouterGroup
	if !viper.GetBool("serve_static") {
		app = engine.Group("")
	} else {
		app = engine.Group("/api")
	}

	{
		auth.Register(app)
		admin.Register(app)
		adapter.Register(app)
		manager.Register(app)
		addition.Register(app)
		conversation.Register(app)
		memory.Register(app)
		billing.Register(app)
	}
}

func main() {
	utils.ReadConf()
	admin.InitInstance()
	channel.InitManager()

	if cli.Run() {
		return
	}

	app := utils.NewEngine()
	worker := middleware.RegisterMiddleware(app)
	defer worker()
	conversation.StartOrphanAttachmentCleanupWorker(connection.DB)

	registerHealthRoutes(app)
	utils.RegisterStaticRoute(app)
	registerApiRouter(app)
	readCorsOrigins()

	if err := app.Run(fmt.Sprintf(":%s", viper.GetString("server.port"))); err != nil {
		panic(err)
	}
}
