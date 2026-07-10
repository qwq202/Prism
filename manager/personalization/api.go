package personalization

import (
	"chat/auth"
	"chat/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type saveForm struct {
	Settings     *Settings `json:"settings" binding:"required"`
	BaseRevision int64     `json:"base_revision"`
}

func LoadAPI(c *gin.Context) {
	user := auth.GetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "user not found"})
		return
	}

	db := utils.GetDBFromContext(c)
	record, err := Load(db, user.GetID(db))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "data": record})
}

func SaveAPI(c *gin.Context) {
	user := auth.GetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "user not found"})
		return
	}

	var form saveForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "invalid form"})
		return
	}

	db := utils.GetDBFromContext(c)
	if form.Settings == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "settings are required"})
		return
	}
	record, err := Save(db, user.GetID(db), *form.Settings, form.BaseRevision)
	if errors.Is(err, ErrRevisionConflict) {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"code":    "revision_conflict",
			"message": ErrRevisionConflict.Error(),
			"data":    record,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": record})
}
