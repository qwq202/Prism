package generation

import (
	"chat/auth"
	"chat/billing"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type WebsocketGenerationForm struct {
	Token  string `json:"token" binding:"required"`
	Prompt string `json:"prompt" binding:"required"`
	Model  string `json:"model" binding:"required"`
}

func ProjectTarDownloadAPI(c *gin.Context) {
	hash := strings.TrimSpace(c.Query("hash"))
	if !utils.IsSha256Hash(hash) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	downloadProjectFile(c, filepath.Join("storage", "generation", hash+".tar.gz"), "code.tar.gz")
}

func ProjectZipDownloadAPI(c *gin.Context) {
	hash := strings.TrimSpace(c.Query("hash"))
	if !utils.IsSha256Hash(hash) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	downloadProjectFile(c, filepath.Join("storage", "generation", hash+".zip"), "code.zip")
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

func generationQuota(buffer *utils.Buffer) float32 {
	if buffer == nil {
		return 0
	}
	return buffer.GetRecordQuota()
}

func settleGenerationQuota(db *sql.DB, cache *redis.Client, user *auth.User, model string, buffer *utils.Buffer, plan bool) {
	if user == nil || buffer == nil || buffer.IsEmpty() {
		return
	}

	quota := buffer.GetRecordQuota()
	if quota <= 0 {
		return
	}

	if plan {
		consumed := auth.FinalizeSubscriptionUsageAmount(db, cache, user, model, quota)
		if consumed+0.0001 < quota {
			user.ChargeQuota(db, quota-consumed)
		}
		return
	}

	user.ChargeQuota(db, quota)
}

func createGenerationBillingRecord(db *sql.DB, user *auth.User, model string, buffer *utils.Buffer) {
	if db == nil || user == nil || buffer == nil || buffer.IsEmpty() {
		return
	}

	billing.CreateRecord(
		db, auth.GetId(db, user), user.Username, "consume",
		buffer.GetTokenName(), model,
		int64(buffer.CountRecordInputToken()), int64(buffer.CountRecordOutputToken()),
		float64(buffer.GetRecordQuota()), buffer.GetDuration(),
		buffer.GetBillingDetail(), buffer.GetRecordPrompts(), buffer.GetRecordResponsePrompts(),
		buffer.GetChannelId(), buffer.GetChannelName(),
	)
}

func GenerateAPI(c *gin.Context) {
	var conn *utils.WebSocket
	if conn = utils.NewWebsocket(c, false); conn == nil {
		return
	}
	defer conn.DeferClose()

	form, err := utils.ReadForm[WebsocketGenerationForm](conn)
	if err != nil {
		return
	}

	user := auth.ParseToken(c, form.Token)

	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	if !auth.HitGroups(db, user, globals.GenerationPermissionGroup) {
		conn.Send(globals.GenerationSegmentResponse{
			Message: "permission denied",
			Quota:   0,
			End:     true,
		})
		return
	}

	check, plan := auth.CanEnableModelWithSubscription(db, cache, user, form.Model, GenerateMessage(form.Prompt))
	if check != nil {
		conn.Send(globals.GenerationSegmentResponse{
			Message: check.Error(),
			Quota:   0,
			End:     true,
		})
		return
	}

	var instance *utils.Buffer
	hash, err := CreateGenerationWithCache(
		auth.GetGroup(db, user),
		form.Model,
		form.Prompt,
		func(buffer *utils.Buffer, data string) {
			instance = buffer
			conn.Send(globals.GenerationSegmentResponse{
				End:     false,
				Message: data,
				Quota:   buffer.GetQuota(),
			})
		},
	)

	if err != nil {
		if instance != nil && instance.HasVisiblePayload() {
			settleGenerationQuota(db, cache, user, form.Model, instance, plan)
			createGenerationBillingRecord(db, user, form.Model, instance)
		} else {
			auth.RevertSubscriptionUsage(db, cache, user, form.Model)
		}
		conn.Send(globals.GenerationSegmentResponse{
			End:   true,
			Error: err.Error(),
			Quota: generationQuota(instance),
		})
		return
	}

	settleGenerationQuota(db, cache, user, form.Model, instance, plan)
	createGenerationBillingRecord(db, user, form.Model, instance)

	conn.Send(globals.GenerationSegmentResponse{
		End:   true,
		Hash:  hash,
		Quota: generationQuota(instance),
	})
}
