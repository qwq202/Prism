package article

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type WebsocketArticleForm struct {
	Token  string `json:"token" binding:"required"`
	Model  string `json:"model" binding:"required"`
	Prompt string `json:"prompt" binding:"required"`
	Title  string `json:"title" binding:"required"`
	Web    bool   `json:"web"`
}

type WebsocketArticleResponse struct {
	Hash string                 `json:"hash"`
	Data StreamProgressResponse `json:"data"`
}

func ProjectTarDownloadAPI(c *gin.Context) {
	hash := strings.TrimSpace(c.Query("hash"))
	if !utils.IsMd5Hash(hash) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	downloadProjectFile(c, filepath.Join("storage", "article", hash+".tar.gz"), "article.tar.gz")
}

func ProjectZipDownloadAPI(c *gin.Context) {
	hash := strings.TrimSpace(c.Query("hash"))
	if !utils.IsMd5Hash(hash) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	downloadProjectFile(c, filepath.Join("storage", "article", hash+".zip"), "article.zip")
}

func downloadProjectFile(c *gin.Context, path string, filename string) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filename)
	c.File(path)
}

func GenerateAPI(c *gin.Context) {
	var conn *utils.WebSocket
	if conn = utils.NewWebsocket(c, false); conn == nil {
		return
	}
	defer conn.DeferClose()

	form, err := utils.ReadForm[WebsocketArticleForm](conn)
	if err != nil {
		return
	}

	user := auth.ParseToken(c, form.Token)
	db := utils.GetDBFromContext(c)

	if !auth.HitGroups(db, user, globals.ArticlePermissionGroup) {
		return
	}

	if len(form.Title) == 0 {
		return
	}

	hash := CreateWorker(c, user, form.Model, form.Prompt, form.Title, form.Web, func(resp StreamProgressResponse) {
		conn.Send(WebsocketArticleResponse{
			Hash: "",
			Data: resp,
		})
	})
	conn.Send(WebsocketArticleResponse{
		Hash: hash,
	})
}
