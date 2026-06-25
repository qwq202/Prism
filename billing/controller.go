package billing

import (
	"chat/admin/analysis"
	"chat/auth"
	"chat/connection"
	"chat/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RecordQueryForm struct {
	UserId      int64  `json:"user_id"`
	Username    string `json:"username"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TokenName   string `json:"token_name"`
	Model       string `json:"model"`
	Type        string `json:"type"`
	ShowChannel bool   `json:"show_channel"`
	Self        bool   `json:"self"`
	CacheHit    bool   `json:"cache_hit"`
}

type RecordStatsForm struct {
	Self bool `json:"self"`
}

type RecordUsageSummaryForm struct {
	UserId    int64  `json:"user_id"`
	Username  string `json:"username"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	TokenName string `json:"token_name"`
	Model     string `json:"model"`
	Self      bool   `json:"self"`
}

func RecordViewAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	isAdmin := user.IsAdmin(db)

	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)

	var form RecordQueryForm
	_ = c.ShouldBindJSON(&form)

	userId := auth.GetId(db, user)
	data, err := ListRecords(db, isAdmin, userId, page, RecordQuery{
		UserId:      form.UserId,
		Username:    form.Username,
		StartTime:   form.StartTime,
		EndTime:     form.EndTime,
		TokenName:   form.TokenName,
		Model:       form.Model,
		Type:        form.Type,
		ShowChannel: form.ShowChannel,
		Self:        form.Self,
		CacheHit:    form.CacheHit,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   data,
	})
}

func RecordUsageSummaryAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	isAdmin := user.IsAdmin(db)
	userId := auth.GetId(db, user)

	var form RecordUsageSummaryForm
	_ = c.ShouldBindJSON(&form)

	data, err := GetRecordUsageSummary(db, isAdmin, userId, RecordQuery{
		UserId:    form.UserId,
		Username:  form.Username,
		StartTime: form.StartTime,
		EndTime:   form.EndTime,
		TokenName: form.TokenName,
		Model:     form.Model,
		Self:      form.Self,
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data":   data,
	})
}

func RecordStatsAPI(c *gin.Context) {
	user := auth.RequireAuth(c)
	if user == nil {
		return
	}

	db := utils.GetDBFromContext(c)
	cache := connection.Cache
	isAdmin := user.IsAdmin(db)
	userId := auth.GetId(db, user)

	var form RecordStatsForm
	_ = c.ShouldBindJSON(&form)

	var rpm, tpm int64
	if isAdmin && !form.Self {
		rpm = analysis.GetRpmToday(cache, "root")
		tpm = analysis.GetTpmToday(cache, "root")
	} else {
		username := utils.GetUserFromContext(c)
		rpm = analysis.GetRpmToday(cache, username)
		tpm = analysis.GetTpmToday(cache, username)
	}

	if !isAdmin || form.Self {
		stats, err := GetUserRecordStats(db, userId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status":  false,
				"message": err.Error(),
			})
			return
		}
		stats.Rpm = rpm
		stats.Tpm = tpm
		c.JSON(http.StatusOK, gin.H{
			"status": true,
			"data":   stats,
		})
		return
	}

	requestData := analysis.GetRequestData(cache)
	var requestToday int64
	if len(requestData.Value) > 0 {
		requestToday = requestData.Value[len(requestData.Value)-1]
	}

	tokenStats, err := GetAllRecordTokenStats(db)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
		"data": RecordStats{
			BillingToday: analysis.GetBillingToday(cache),
			BillingMonth: analysis.GetBillingMonth(cache),
			RequestToday: requestToday,
			RequestMonth: 0,
			Rpm:          rpm,
			Tpm:          tpm,
			TotalTokens:  tokenStats.TotalTokens,
		},
	})
}
