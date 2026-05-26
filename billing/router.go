package billing

import "github.com/gin-gonic/gin"

func Register(app *gin.RouterGroup) {
	app.POST("/record/view", RecordViewAPI)
	app.POST("/record/stats", RecordStatsAPI)
	app.POST("/record/usage-summary", RecordUsageSummaryAPI)
}
