package manager

import (
	"chat/admin"
	"chat/billing"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
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

func parseMetricModels(value string) []string {
	seen := make(map[string]bool)
	models := make([]string, 0)

	for _, item := range strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t'
	}) {
		model := strings.TrimSpace(item)
		if model == "" || seen[model] {
			continue
		}

		seen[model] = true
		models = append(models, model)
	}

	return models
}

func ModelMetricsAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)

	if models := parseMetricModels(c.Query("models")); len(models) > 0 {
		data := make(map[string]billing.ModelUsageStats, len(models))
		for _, model := range models {
			stats, err := billing.GetModelUsageStats(utils.GetDBFromContext(c), model, 24)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"status":  false,
					"message": err.Error(),
				})
				return
			}
			data[model] = stats
		}

		c.JSON(http.StatusOK, gin.H{
			"status": true,
			"data":   data,
		})
		return
	}

	model := strings.TrimSpace(c.Query("model"))
	if model == "" {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": "model is required",
		})
		return
	}

	stats, err := billing.GetModelUsageStats(utils.GetDBFromContext(c), model, 24)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   stats,
	})
}

func ChargeAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)
	c.JSON(http.StatusOK, channel.ChargeInstance.ListRules())
}

func PlanAPI(c *gin.Context) {
	setFreshV1ResponseHeaders(c)
	c.JSON(http.StatusOK, channel.PlanInstance.GetPlans())
}
