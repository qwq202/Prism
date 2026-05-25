package admin

import (
	"chat/globals"
	"chat/utils"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AttachmentFile struct {
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	UpdatedAt      string `json:"updated_at"`
	StorageMode    string `json:"storage_mode"`
	PublicURL      string `json:"public_url"`
	Referenced     bool   `json:"referenced"`
	ReferenceCount int64  `json:"reference_count"`
}

func attachmentDeleteForced(c *gin.Context) bool {
	force := strings.ToLower(strings.TrimSpace(c.Query("force")))
	return force == "true" || force == "1"
}

func loadAttachmentReferenceCount(db *sql.DB) map[string]int64 {
	rows, err := globals.QueryDb(db, `SELECT data FROM conversation`)
	if err != nil {
		return map[string]int64{}
	}
	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	references := map[string]int64{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			continue
		}

		for _, name := range utils.ExtractAttachmentNames(data) {
			references[name]++
		}
	}

	return references
}

func ListAttachments(db *sql.DB) ([]AttachmentFile, error) {
	items, err := utils.ListConfiguredStoredAttachments()
	if err != nil {
		return nil, err
	}

	references := loadAttachmentReferenceCount(db)
	result := make([]AttachmentFile, 0, len(items))
	for _, item := range items {
		count := references[item.Name]
		result = append(result, AttachmentFile{
			Name:           item.Name,
			Size:           item.Size,
			UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
			StorageMode:    strings.ToLower(strings.TrimSpace(item.StorageMode)),
			PublicURL:      item.PublicURL,
			Referenced:     count > 0,
			ReferenceCount: count,
		})
	}

	return result, nil
}

func ListAttachmentAPI(c *gin.Context) {
	data, err := ListAttachments(utils.GetDBFromContext(c))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

func DeleteAttachmentAPI(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "attachment name is required",
		})
		return
	}

	referenceCount := loadAttachmentReferenceCount(utils.GetDBFromContext(c))[name]
	if referenceCount > 0 && !attachmentDeleteForced(c) {
		c.JSON(http.StatusOK, gin.H{
			"status":          false,
			"error":           "attachment is still referenced",
			"referenced":      true,
			"reference_count": referenceCount,
		})
		return
	}

	if err := utils.DeleteConfiguredStoredAttachment(name); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}
