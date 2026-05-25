package memory

import "github.com/gin-gonic/gin"

func Register(app *gin.RouterGroup) {
	router := app.Group("/memory")
	{
		router.GET("/list", ListAPI)
		router.POST("/create", CreateAPI)
		router.POST("/update", UpdateAPI)
		router.POST("/delete", DeleteAPI)
		router.GET("/delete", DeleteAPI)
	}
}
