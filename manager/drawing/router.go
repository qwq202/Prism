package drawing

import "github.com/gin-gonic/gin"

func Register(app *gin.RouterGroup) {
	router := app.Group("/drawing")
	{
		router.GET("/workspaces", LoadWorkspaceAPI)
		router.POST("/workspaces", SaveWorkspaceAPI)
		router.GET("/tasks", ListTasksAPI)
		router.POST("/tasks", CreateTaskAPI)
		router.GET("/tasks/:id", GetTaskAPI)
		router.POST("/tasks/:id/cancel", CancelTaskAPI)
	}
}
