package broadcast

import "github.com/gin-gonic/gin"

func getLatestBroadcast(c *gin.Context) *Broadcast {
	return getLatestActiveBroadcast(c)
}
