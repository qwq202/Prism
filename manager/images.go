package manager

import (
	"chat/auth"
	"chat/globals"
	"chat/utils"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	maxAttachmentUploadBytes          int64 = 100 * 1024 * 1024
	maxAttachmentMultipartOverhead    int64 = 1024 * 1024
	maxAttachmentMultipartUploadBytes       = maxAttachmentUploadBytes + maxAttachmentMultipartOverhead
	attachmentUploadServerError             = "failed to upload attachment"
)

type AttachmentUploadResponse struct {
	Status bool   `json:"status"`
	Url    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
}

func attachmentUploadSizeError() string {
	return fmt.Sprintf("file exceeds %dMB upload limit", maxAttachmentUploadBytes/1024/1024)
}

func attachmentUploadError(c *gin.Context, error string) {
	c.JSON(http.StatusOK, AttachmentUploadResponse{
		Status: false,
		Error:  error,
	})
}

func attachmentUploadInternalError(c *gin.Context, step string, err error) {
	globals.Warn(fmt.Sprintf("[attachment] %s failed: %s", step, err.Error()))
	attachmentUploadError(c, attachmentUploadServerError)
}

func isAttachmentBodyTooLarge(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "request body too large")
}

func isSupportedAttachmentContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}

	mediaType = strings.ToLower(mediaType)
	return strings.HasPrefix(mediaType, "image/") && mediaType != "image/svg+xml"
}

func isGenericAttachmentContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}

	mediaType = strings.ToLower(mediaType)
	return mediaType == "" || mediaType == "application/octet-stream"
}

func UploadAttachmentAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	if c.Request.ContentLength > maxAttachmentMultipartUploadBytes {
		attachmentUploadError(c, attachmentUploadSizeError())
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAttachmentMultipartUploadBytes)

	file, err := c.FormFile("file")
	if err != nil {
		if isAttachmentBodyTooLarge(err) {
			attachmentUploadError(c, attachmentUploadSizeError())
			return
		}
		attachmentUploadError(c, "file is required")
		return
	}

	if file.Size > maxAttachmentUploadBytes {
		attachmentUploadError(c, attachmentUploadSizeError())
		return
	}
	if file.Size <= 0 {
		attachmentUploadError(c, "file is empty")
		return
	}

	headerContentType := file.Header.Get("Content-Type")
	if headerContentType != "" && !isGenericAttachmentContentType(headerContentType) && !isSupportedAttachmentContentType(headerContentType) {
		attachmentUploadError(c, "only raster image upload is supported")
		return
	}

	src, err := file.Open()
	if err != nil {
		attachmentUploadInternalError(c, "open upload", err)
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		if isAttachmentBodyTooLarge(err) {
			attachmentUploadError(c, attachmentUploadSizeError())
			return
		}
		attachmentUploadInternalError(c, "read upload", err)
		return
	}

	detectedContentType := http.DetectContentType(data)
	if !isSupportedAttachmentContentType(detectedContentType) {
		attachmentUploadError(c, "invalid image data")
		return
	}

	url, err := utils.StoreAttachmentData(file.Filename, data, detectedContentType)
	if err != nil {
		attachmentUploadInternalError(c, "store upload", err)
		return
	}

	c.JSON(http.StatusOK, AttachmentUploadResponse{
		Status: true,
		Url:    url,
	})
}
