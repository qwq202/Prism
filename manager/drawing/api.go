package drawing

import (
	"chat/auth"
	"chat/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func LoadWorkspaceAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	state, err := LoadWorkspaceState(db, user.GetID(db))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": state})
}

func SaveWorkspaceAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	var form saveWorkspaceForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "invalid form"})
		return
	}

	db := utils.GetDBFromContext(c)
	state, err := SaveWorkspaceState(db, user.GetID(db), WorkspaceState{
		ActiveWorkspaceID: form.ActiveWorkspaceID,
		Workspaces:        form.Workspaces,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": state})
}

func ListTasksAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	tasks, err := LoadActiveTasks(db, user.GetID(db))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": tasks})
}

func CreateTaskAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	var form createTaskForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "invalid form"})
		return
	}

	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)
	userID := user.GetID(db)
	task, err := CreateTask(db, userID, form)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	StartTask(db, cache, userID, task.TaskID)
	c.JSON(http.StatusOK, gin.H{"status": true, "data": task})
}

func GetTaskAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	task, err := LoadTask(db, user.GetID(db), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": task})
}

func CancelTaskAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	task, err := MarkTaskCanceled(db, user.GetID(db), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": task})
}
