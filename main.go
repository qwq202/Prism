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
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

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

	utils.RegisterStaticRoute(app)
	registerApiRouter(app)
	readCorsOrigins()

	if err := app.Run(fmt.Sprintf(":%s", viper.GetString("server.port"))); err != nil {
		panic(err)
	}
}
