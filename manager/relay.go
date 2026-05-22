package manager

import (
	"chat/admin"
	"chat/channel"
	"chat/globals"
	"github.com/gin-gonic/gin"
	"net/http"
)

func setFreshV1ResponseHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

func ModelAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)
	c.JSON(http.StatusOK, globals.V1ListModels)
}

func MarketAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)
	c.JSON(http.StatusOK, admin.MarketInstance.GetViewModels())
}

func ChargeAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)
	c.JSON(http.StatusOK, channel.ChargeInstance.ListRules())
}

func PlanAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)
	c.JSON(http.StatusOK, channel.PlanInstance.GetPlans())
}
