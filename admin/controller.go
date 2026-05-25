package admin

import (
	"chat/admin/analysis"
	"chat/auth"
	"chat/channel"
	"chat/utils"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type GenerateInvitationForm struct {
	Type   string  `json:"type"`
	Quota  float32 `json:"quota"`
	Number int     `json:"number"`
}

type DeleteInvitationForm struct {
	Code string `json:"code"`
}

type GenerateRedeemForm struct {
	Quota  float32 `json:"quota"`
	Number int     `json:"number"`
}

type PasswordMigrationForm struct {
	Id       int64  `json:"id"`
	Password string `json:"password"`
}

type CreateUserForm struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type EmailMigrationForm struct {
	Id    int64  `json:"id"`
	Email string `json:"email"`
}

type SetAdminForm struct {
	Id    int64 `json:"id"`
	Admin bool  `json:"admin"`
}

type BanForm struct {
	Id  int64 `json:"id"`
	Ban bool  `json:"ban"`
}

type DeleteUserForm struct {
	Id int64 `json:"id" binding:"required"`
}

type QuotaOperationForm struct {
	Id       int64    `json:"id" binding:"required"`
	Quota    *float32 `json:"quota" binding:"required"`
	Override bool     `json:"override"`
}

type SubscriptionOperationForm struct {
	Id      int64  `json:"id" binding:"required"`
	Expired string `json:"expired" binding:"required"`
}

type SubscriptionLevelForm struct {
	Id    int64  `json:"id" binding:"required"`
	Level *int64 `json:"level" binding:"required"`
}

type ReleaseUsageForm struct {
	Id   int64  `json:"id" binding:"required"`
	Type string `json:"type"`
}

type UpdateRootPasswordForm struct {
	Password string `json:"password" binding:"required"`
}

func freshTokenForCurrentUserID(db *sql.DB, username string, id int64) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || id <= 0 {
		return "", nil
	}

	user := auth.GetUserByName(db, username)
	if user == nil || user.ID != id {
		return "", nil
	}

	return user.GenerateTokenSafe(db)
}

func currentUserMatchesID(c *gin.Context, db *sql.DB, id int64) bool {
	if id <= 0 {
		return false
	}

	username := strings.TrimSpace(utils.GetUserFromContext(c))
	if username == "" {
		return false
	}

	current := auth.GetUserByName(db, username)
	return current != nil && current.ID == id
}

func UpdateMarketAPI(c *gin.Context) {
	var form MarketModelList
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	err := MarketInstance.SetModels(form)
	if err != nil {
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

func InfoAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	c.JSON(http.StatusOK, InfoForm{
		OnlineChats:       utils.GetConns(),
		SubscriptionCount: analysis.GetSubscriptionUsers(db),
		BillingToday:      analysis.GetBillingToday(cache),
		BillingMonth:      analysis.GetBillingMonth(cache),
		BillingYesterday:  analysis.GetBillingYesterday(cache),
		BillingLastMonth:  analysis.GetBillingLastMonth(cache),
	})
}

func ModelAnalysisAPI(c *gin.Context) {
	cache := utils.GetCacheFromContext(c)
	c.JSON(http.StatusOK, analysis.GetSortedModelData(cache))
}

func RequestAnalysisAPI(c *gin.Context) {
	cache := utils.GetCacheFromContext(c)
	c.JSON(http.StatusOK, analysis.GetRequestData(cache))
}

func BillingAnalysisAPI(c *gin.Context) {
	cache := utils.GetCacheFromContext(c)
	c.JSON(http.StatusOK, analysis.GetBillingData(cache))
}

func ErrorAnalysisAPI(c *gin.Context) {
	cache := utils.GetCacheFromContext(c)
	c.JSON(http.StatusOK, analysis.GetErrorData(cache))
}

func UserTypeAnalysisAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	if form, err := analysis.GetUserTypeData(db); err != nil {
		c.JSON(http.StatusOK, &analysis.UserTypeForm{})
	} else {
		c.JSON(http.StatusOK, form)
	}
}

type ChannelAnalysisRequest struct {
	ChannelIds []int `json:"channel_ids"`
}

func ChannelAnalysisAPI(c *gin.Context) {
	cache := utils.GetCacheFromContext(c)

	var req ChannelAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.ChannelIds) == 0 {
		// fall back to all channels in conduit
		seq := channel.ConduitInstance.Sequence
		ids := make([]int, 0, len(seq))
		for _, ch := range seq {
			ids = append(ids, ch.GetId())
		}
		req.ChannelIds = ids
	}

	c.JSON(http.StatusOK, analysis.GetChannelStats(cache, req.ChannelIds))
}

func ActiveUserAnalysisAPI(c *gin.Context) {
	cache := utils.GetCacheFromContext(c)
	c.JSON(http.StatusOK, analysis.GetActiveUserData(cache))
}

func RegistrationAnalysisAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	c.JSON(http.StatusOK, analysis.GetRegistrationData(db))
}

func ConversionFunnelAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	c.JSON(http.StatusOK, analysis.GetConversionFunnel(db))
}

func parsePageQuery(c *gin.Context) (int64, bool) {
	raw := strings.TrimSpace(c.Query("page"))
	if raw == "" {
		return 0, true
	}

	page, err := strconv.Atoi(raw)
	if err != nil || page < 0 {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  "invalid page",
		})
		return 0, false
	}

	return int64(page), true
}

func RedeemListAPI(c *gin.Context) {
	page, ok := parsePageQuery(c)
	if !ok {
		return
	}

	db := utils.GetDBFromContext(c)
	c.JSON(http.StatusOK, GetRedeemData(db, page))
}

func DeleteRedeemAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form DeleteInvitationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	err := DeleteRedeemCode(db, form.Code)
	c.JSON(http.StatusOK, gin.H{
		"status": err == nil,
		"error":  err,
	})
}

func RedeemBatchListAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	c.JSON(http.StatusOK, GetRedeemBatches(db))
}

func RedeemBatchCodesAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	batchId := c.Param("id")
	c.JSON(http.StatusOK, GetBatchCodes(db, batchId))
}

func InvitationPaginationAPI(c *gin.Context) {
	page, ok := parsePageQuery(c)
	if !ok {
		return
	}

	db := utils.GetDBFromContext(c)

	c.JSON(http.StatusOK, GetInvitationPagination(db, page))
}

func DeleteInvitationAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form DeleteInvitationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	err := DeleteInvitationCode(db, form.Code)
	c.JSON(http.StatusOK, gin.H{
		"status": err == nil,
		"error":  err,
	})
}
func GenerateInvitationAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form GenerateInvitationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateInvitations(db, form.Number, form.Quota, form.Type))
}

func GenerateRedeemAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form GenerateRedeemForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, GenerateRedeemCodes(db, form.Number, form.Quota))
}

func UserPaginationAPI(c *gin.Context) {
	page, ok := parsePageQuery(c)
	if !ok {
		return
	}

	db := utils.GetDBFromContext(c)

	search := strings.TrimSpace(c.Query("search"))
	c.JSON(http.StatusOK, getUsersForm(db, page, search))
}

func CreateUserAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form CreateUserForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	if err := createUser(db, form.Username, form.Email, form.Password); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func UpdatePasswordAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	var form PasswordMigrationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	err := passwordMigration(db, cache, form.Id, form.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	resp := gin.H{
		"status": true,
	}
	token, err := freshTokenForCurrentUserID(db, utils.GetUserFromContext(c), form.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}
	if token != "" {
		resp["token"] = token
	}

	c.JSON(http.StatusOK, resp)
}

func UpdateEmailAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form EmailMigrationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	err := emailMigration(db, form.Id, form.Email)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func SetAdminAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form SetAdminForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	if currentUserMatchesID(c, db, form.Id) {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": "cannot change current user admin status",
		})
		return
	}

	err := setAdmin(db, form.Id, form.Admin)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func BanAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form BanForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	if form.Ban && currentUserMatchesID(c, db, form.Id) {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": "cannot ban current user",
		})
		return
	}

	err := banUser(db, form.Id, form.Ban)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func DeleteUserAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	var form DeleteUserForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	current := auth.GetUserByCtx(c)
	if current == nil {
		return
	}
	if auth.GetId(db, current) == form.Id {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": "cannot delete current user",
		})
		return
	}

	if err := deleteUser(db, cache, form.Id); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func UserQuotaAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form QuotaOperationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	err := quotaMigration(db, form.Id, *form.Quota, form.Override)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

type BatchUserForm struct {
	Ids    []int64  `json:"ids"`
	Action string   `json:"action"`
	Value  *float32 `json:"value"`
}

func BatchUserAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form BatchUserForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	if len(form.Ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "no users selected"})
		return
	}

	if form.Action == "ban" {
		for _, id := range form.Ids {
			if currentUserMatchesID(c, db, id) {
				c.JSON(http.StatusOK, gin.H{"status": false, "message": "cannot ban current user"})
				return
			}
		}
	}

	quota := float32(0)
	if form.Value != nil {
		quota = *form.Value
	}

	if err := batchUsers(db, form.Ids, form.Action, quota); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": true})
}

func UserSubscriptionAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form SubscriptionOperationForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	// convert to time
	if _, err := time.Parse("2006-01-02 15:04:05", form.Expired); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	if err := subscriptionMigration(db, form.Id, form.Expired); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func SubscriptionLevelAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)

	var form SubscriptionLevelForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	err := subscriptionLevelMigration(db, form.Id, *form.Level)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func ReleaseUsageAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)

	var form ReleaseUsageForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	err := releaseUsage(db, cache, form.Id, form.Type)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": true,
	})
}

func UpdateRootPasswordAPI(c *gin.Context) {
	var form UpdateRootPasswordForm
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)
	err := UpdateRootPassword(db, cache, form.Password)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": false,
			"error":  err.Error(),
		})
		return
	}

	resp := gin.H{
		"status": true,
	}
	if strings.TrimSpace(utils.GetUserFromContext(c)) == "root" {
		token, err := (&auth.User{Username: "root"}).GenerateTokenSafe(db)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"status": false,
				"error":  err.Error(),
			})
			return
		}
		resp["token"] = token
	}

	c.JSON(http.StatusOK, resp)
}

func ListLoggerAPI(c *gin.Context) {
	c.JSON(http.StatusOK, ListLogs())
}

func DownloadLoggerAPI(c *gin.Context) {
	path := c.Query("path")
	if err := getBlobFile(c, path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": false,
			"error":  err.Error(),
		})
	}
}

func DeleteLoggerAPI(c *gin.Context) {
	path := c.Query("path")
	if err := deleteLogFile(path); err != nil {
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

func ConsoleLoggerAPI(c *gin.Context) {
	n := utils.ParseInt(c.Query("n"))

	content := getLatestLogs(n)

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"content": content,
	})
}

func PaymentOrderListAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	page, _ := strconv.ParseInt(c.Query("page"), 10, 64)
	search := strings.TrimSpace(c.Query("search"))
	c.JSON(http.StatusOK, getPaymentOrdersForm(db, page, search))
}

func PaymentOrderRecheckAPI(c *gin.Context) {
	db := utils.GetDBFromContext(c)
	orderId := c.Query("order")
	if orderId == "" {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "order id is required"})
		return
	}

	ok, state, err := recheckPaymentOrder(db, orderId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      ok,
		"order_state": state,
		"is_changed":  false,
	})
}

type WarmupForm struct {
	Urls []string `json:"urls" binding:"required"`
}

const (
	maxWarmupUrls        = 100
	maxWarmupURLLength   = 2048
	maxWarmupConcurrency = 8
)

type WarmupResult struct {
	Url    string `json:"url"`
	Status int    `json:"status"`
	Error  string `json:"error,omitempty"`
}

func isUnsafeWarmupAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() {
		return true
	}

	if addr.IsUnspecified() ||
		addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsMulticast() {
		return true
	}

	if addr.Is4() {
		octets := addr.As4()
		if octets[0] == 100 && octets[1] >= 64 && octets[1] <= 127 {
			return true
		}
	}

	return false
}

func isUnsafeWarmupHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.Contains(host, "%") {
		return true
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return isUnsafeWarmupAddr(addr)
	}

	return false
}

func validateWarmupURL(target string) (*neturl.URL, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("url is required")
	}
	if len(target) > maxWarmupURLLength {
		return nil, fmt.Errorf("url is too long")
	}

	parsed, err := neturl.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("url credentials are not allowed")
	}
	if isUnsafeWarmupHost(parsed.Hostname()) {
		return nil, fmt.Errorf("unsupported url host")
	}

	return parsed, nil
}

func warmupDialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if isUnsafeWarmupHost(host) {
		return nil, fmt.Errorf("unsupported url host")
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("url host has no ip addresses")
	}

	target := ""
	for _, ip := range addrs {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok || isUnsafeWarmupAddr(addr) {
			return nil, fmt.Errorf("unsupported url host")
		}

		addr = addr.Unmap()
		if target != "" {
			continue
		}
		if network == "tcp4" && !addr.Is4() {
			continue
		}
		if network == "tcp6" && !addr.Is6() {
			continue
		}
		target = net.JoinHostPort(addr.String(), port)
	}
	if target == "" {
		return nil, fmt.Errorf("url host has no compatible ip addresses")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, target)
}

func warmupHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: warmupDialContext,
		},
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			_, err := validateWarmupURL(req.URL.String())
			return err
		},
	}
}

func warmupUrl(target string) WarmupResult {
	parsed, err := validateWarmupURL(target)
	if err != nil {
		return WarmupResult{Url: strings.TrimSpace(target), Status: 0, Error: err.Error()}
	}

	resp, err := warmupHTTPClient().Get(parsed.String())
	if err != nil {
		return WarmupResult{Url: parsed.String(), Status: 0, Error: err.Error()}
	}
	defer resp.Body.Close()
	return WarmupResult{Url: parsed.String(), Status: resp.StatusCode}
}

func WarmupAPI(c *gin.Context) {
	var form WarmupForm
	if err := c.ShouldBindJSON(&form); err != nil || len(form.Urls) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "urls are required"})
		return
	}
	if len(form.Urls) > maxWarmupUrls {
		c.JSON(http.StatusOK, gin.H{"status": false, "message": "too many urls"})
		return
	}

	results := make([]WarmupResult, len(form.Urls))
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWarmupConcurrency)

	for i, url := range form.Urls {
		sem <- struct{}{}
		wg.Add(1)
		go func(idx int, target string) {
			defer func() {
				<-sem
				wg.Done()
			}()
			results[idx] = warmupUrl(target)
		}(i, url)
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"status":  true,
		"results": results,
	})
}
