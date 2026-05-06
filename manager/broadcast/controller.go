package broadcast

import (
	"chat/auth"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func ViewBroadcastAPI(c *gin.Context) {
	c.JSON(http.StatusOK, getLatestBroadcast(c))
}

func CreateBroadcastAPI(c *gin.Context) {
	user := auth.RequireAdmin(c)
	if user == nil {
		return
	}

	var form createRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, createResponse{
			Status: false,
			Error:  err.Error(),
		})
		return
	}

	err := createBroadcast(c, user, form)
	if err != nil {
		c.JSON(http.StatusOK, createResponse{
			Status: false,
			Error:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, createResponse{
		Status: true,
	})
}

func UpdateBroadcastAPI(c *gin.Context) {
	user := auth.RequireAdmin(c)
	if user == nil {
		return
	}

	var form updateRequest
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, createResponse{Status: false, Error: err.Error()})
		return
	}

	err := updateBroadcast(c, form)
	c.JSON(http.StatusOK, createResponse{Status: err == nil, Error: func() string {
		if err != nil {
			return err.Error()
		}
		return ""
	}()})
}

func RemoveBroadcastAPI(c *gin.Context) {
	user := auth.RequireAdmin(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, createResponse{Status: false, Error: "invalid id"})
		return
	}

	err = removeBroadcast(c, id)
	c.JSON(http.StatusOK, createResponse{Status: err == nil, Error: func() string {
		if err != nil {
			return err.Error()
		}
		return ""
	}()})
}

func GetBroadcastListAPI(c *gin.Context) {
	user := auth.RequireAdmin(c)
	if user == nil {
		return
	}

	data, err := getBroadcastList(c)
	if err != nil {
		c.JSON(http.StatusOK, listResponse{
			Data: []Info{},
		})
		return
	}

	c.JSON(http.StatusOK, listResponse{
		Data: data,
	})
}
