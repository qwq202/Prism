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
