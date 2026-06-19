package addition

import (
	"chat/addition/article"
	"chat/addition/card"
	"github.com/gin-gonic/gin"
)

func Register(app *gin.RouterGroup) {
	{
		app.POST("/card", card.HandlerAPI)

		app.GET("/article/create", article.GenerateAPI)
		app.GET("/article/download/tar", article.ProjectTarDownloadAPI)
		app.GET("/article/download/zip", article.ProjectZipDownloadAPI)
	}
}
