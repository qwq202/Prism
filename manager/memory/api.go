package memory

import (
	"chat/auth"
	"chat/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type createForm struct {
	Content  string `json:"content" binding:"required"`
	Category string `json:"category"`
}

type updateForm struct {
	ID       int64  `json:"id" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Category string `json:"category"`
}

type deleteForm struct {
	ID int64 `json:"id"`
}

func getDeleteID(c *gin.Context) (int64, error) {
	if c.Request.Method == http.MethodPost {
		var form deleteForm
		if err := c.ShouldBindJSON(&form); err == nil {
			return form.ID, nil
		}
	}

	return strconv.ParseInt(c.Query("id"), 10, 64)
}

func ListAPI(c *gin.Context) {
	user := auth.GetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "user not found"})
		return
	}

	db := utils.GetDBFromContext(c)
	memories, err := ListByUser(db, user.GetID(db), strings.TrimSpace(c.Query("q")), DefaultMemoryLimit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": memories})
}

func CreateAPI(c *gin.Context) {
	user := auth.GetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "user not found"})
		return
	}

	var form createForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "invalid form"})
		return
	}

	if containsSensitiveContent(form.Content) {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "sensitive content cannot be stored"})
		return
	}

	db := utils.GetDBFromContext(c)
	record, err := Create(db, user.GetID(db), form.Content, SourceManual, form.Category)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": record})
}

func UpdateAPI(c *gin.Context) {
	user := auth.GetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "user not found"})
		return
	}

	var form updateForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "invalid form"})
		return
	}

	if containsSensitiveContent(form.Content) {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "sensitive content cannot be stored"})
		return
	}

	db := utils.GetDBFromContext(c)
	record, err := Update(db, user.GetID(db), form.ID, form.Content, form.Category)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true, "data": record})
}

func DeleteAPI(c *gin.Context) {
	user := auth.GetUser(c)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "user not found"})
		return
	}

	id, err := getDeleteID(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "invalid id"})
		return
	}

	db := utils.GetDBFromContext(c)
	if err := Delete(db, user.GetID(db), id); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}
