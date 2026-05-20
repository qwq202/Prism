package manager

import (
	"chat/auth"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
)

func VideoContentAPI(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			globals.Warn(fmt.Sprintf("caught panic from videos content api: %s (client: %s)\n%s",
				err, c.ClientIP(), stack,
			))
		}
	}()

	db := utils.GetDBFromContext(c)

	username := utils.GetUserFromContext(c)
	if username == "" || utils.GetAgentFromContext(c) != "token" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "access denied",
		})
		return
	}

	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "video id is required",
		})
		return
	}

	user := &auth.User{Username: username}
	userId := user.GetID(db)
	var jobModel interface{}

	err := globals.QueryRowDb(db, `
		SELECT model
		FROM conversation
		WHERE user_id = ? AND task_id = ?
		LIMIT 1
	`, userId, id).Scan(&jobModel)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("cannot find video job for video id %s", id),
		})
		return
	}

	var model string
	if value, ok := jobModel.([]byte); ok {
		model = string(value)
	} else if str, ok := jobModel.(string); ok {
		model = str
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("cannot parse model from conversation for video id %s", id),
		})
		return
	}

	group := auth.GetGroup(db, user)
	ticker := channel.ConduitInstance.GetTicker(model, group)
	if ticker == nil || ticker.IsEmpty() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("cannot find channel for model %s", model),
		})
		return
	}

	var lastErr error
	for !ticker.IsDone() {
		ch := ticker.Next()
		if ch == nil {
			break
		}

		endpoint := ch.GetEndpoint()
		secret := ch.GetRandomSecret()
		uri := fmt.Sprintf("%s/v1/videos/%s/content", endpoint, id)

		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", secret),
		}

		data, err := utils.HttpRaw(uri, http.MethodGet, headers, nil, []globals.ProxyConfig{ch.GetProxy()})
		if err != nil || data == nil {
			lastErr = err
			continue
		}

		c.Data(http.StatusOK, "video/mp4", data)
		return
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("failed to fetch video content")
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error": lastErr.Error(),
	})
}
