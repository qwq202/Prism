package drawing

import "github.com/gin-gonic/gin"

func Register(app *gin.RouterGroup) {
	router := app.Group("/drawing")
	{
		router.GET("/workspaces", LoadWorkspaceAPI)
		router.POST("/workspaces", SaveWorkspaceAPI)
	}
}
