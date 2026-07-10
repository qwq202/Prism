package personalization

import "github.com/gin-gonic/gin"

func Register(app *gin.RouterGroup) {
	app.GET("/personalization", LoadAPI)
	app.POST("/personalization", SaveAPI)
}
