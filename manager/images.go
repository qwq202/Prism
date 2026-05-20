package manager

import (
	"chat/auth"
	"chat/utils"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type AttachmentUploadResponse struct {
	Status bool   `json:"status"`
	Url    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

func UploadAttachmentAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, AttachmentUploadResponse{
			Status: false,
			Error:  "file is required",
		})
		return
	}

	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(file.Header.Get("Content-Type"))), "image/") {
		c.JSON(http.StatusOK, AttachmentUploadResponse{
			Status: false,
			Error:  "only image upload is supported",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusOK, AttachmentUploadResponse{
			Status: false,
			Error:  err.Error(),
		})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusOK, AttachmentUploadResponse{
			Status: false,
			Error:  err.Error(),
		})
		return
	}

	url, err := utils.StoreAttachmentData(file.Filename, data, file.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusOK, AttachmentUploadResponse{
			Status: false,
			Error:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, AttachmentUploadResponse{
		Status: true,
		Url:    url,
	})
}
