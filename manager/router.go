package manager

import (
	"chat/manager/broadcast"

	"github.com/gin-gonic/gin"
)

func Register(app *gin.RouterGroup) {
	app.GET("/chat", ChatAPI)
	app.GET("/v1/models", ModelAPI)
	app.GET("/v1/market", MarketAPI)
	app.GET("/v1/charge", ChargeAPI)
	app.GET("/v1/plans", PlanAPI)
	app.POST("/attachment/upload", UploadAttachmentAPI)
	app.GET("/dashboard/billing/usage", GetBillingUsage)
	app.GET("/dashboard/billing/subscription", GetSubscription)
	app.GET("/videos/:id/content", VideoContentAPI)

	broadcast.Register(app)
}
