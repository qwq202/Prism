package manager

import (
	"chat/auth"
	"chat/channel"
	"chat/connection"
	"chat/globals"
	"chat/manager/conversation"
	"chat/utils"
	"database/sql"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
)

type chatTestCharge struct{}

func (chatTestCharge) GetType() string             { return globals.TimesBilling }
func (chatTestCharge) GetModels() []string         { return nil }
func (chatTestCharge) GetInput() float32           { return 0 }
func (chatTestCharge) GetOutput() float32          { return 9 }
func (chatTestCharge) SupportAnonymous() bool      { return true }
func (chatTestCharge) IsBilling() bool             { return true }
func (chatTestCharge) IsBillingType(t string) bool { return t == globals.TimesBilling }
func (chatTestCharge) GetLimit() float32           { return 9 }

type chatTokenTestCharge struct{}

func (chatTokenTestCharge) GetType() string             { return globals.TokenBilling }
func (chatTokenTestCharge) GetModels() []string         { return nil }
func (chatTokenTestCharge) GetInput() float32           { return 0 }
func (chatTokenTestCharge) GetOutput() float32          { return 1 }
func (chatTokenTestCharge) SupportAnonymous() bool      { return true }
func (chatTokenTestCharge) IsBilling() bool             { return true }
func (chatTokenTestCharge) IsBillingType(t string) bool { return t == globals.TokenBilling }
func (chatTokenTestCharge) GetLimit() float32           { return 1 }

func openChatQuotaTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previousSqlite := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previousSqlite
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "chat-quota.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreateQuotaTable(db)
	connection.CreateSubscriptionTable(db)

	return db
}

func newCollectQuotaTestContext(t *testing.T, db *sql.DB) *gin.Context {
	t.Helper()

	cache := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		MaxRetries:   -1,
		DialTimeout:  time.Millisecond,
		ReadTimeout:  time.Millisecond,
		WriteTimeout: time.Millisecond,
	})
	t.Cleanup(func() {
		_ = cache.Close()
	})

	return newCollectQuotaTestContextWithCache(t, db, cache)
}

func newCollectQuotaTestContextWithCache(t *testing.T, db *sql.DB, cache *redis.Client) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("db", db)
	c.Set("cache", cache)

	return c
}

func useChatTestChargeInstance(t *testing.T) {
	t.Helper()

	previousCharge := channel.ChargeInstance
	channel.ChargeInstance = &channel.ChargeManager{
		Models: map[string]*channel.Charge{
			globals.GPT3Turbo: {
				Type:   globals.TimesBilling,
				Models: []string{globals.GPT3Turbo},
				Output: 9,
			},
		},
	}
	t.Cleanup(func() {
		channel.ChargeInstance = previousCharge
	})
}

func useChatTokenTestChargeInstance(t *testing.T) {
	t.Helper()

	previousCharge := channel.ChargeInstance
	channel.ChargeInstance = &channel.ChargeManager{
		Models: map[string]*channel.Charge{
			globals.GPT3Turbo: {
				Type:   globals.TokenBilling,
				Models: []string{globals.GPT3Turbo},
				Input:  0,
				Output: 1,
			},
		},
	}
	t.Cleanup(func() {
		channel.ChargeInstance = previousCharge
	})
}

func TestLatestMessageContentHandlesEmptySegment(t *testing.T) {
	if content, ok := latestMessageContent(nil); ok || content != "" {
		t.Fatalf("expected empty segment to be rejected, got content=%q ok=%v", content, ok)
	}

	content, ok := latestMessageContent([]globals.Message{
		{Role: globals.User, Content: "first"},
		{Role: globals.User, Content: "latest"},
	})
	if !ok || content != "latest" {
		t.Fatalf("expected latest message content, got content=%q ok=%v", content, ok)
	}
}

func TestExtractAssistantMessageFromBufferPersistsBillingMetadata(t *testing.T) {
	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	buffer.Write("hello")

	message := extractAssistantMessageFromBuffer(buffer, false, true)
	if message.Quota != 9 {
		t.Fatalf("expected quota 9 to be persisted, got %f", message.Quota)
	}
	if !message.Plan {
		t.Fatalf("expected plan billing marker to be persisted")
	}
}

func TestCollectQuotaDrainsBalanceWithoutDebtWhenFinalCostExceedsBalance(t *testing.T) {
	db := openChatQuotaTestDB(t)
	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 1) {
		t.Fatalf("set quota")
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	buffer.Write("hello")

	CollectQuota(newCollectQuotaTestContext(t, db), user, buffer, false, nil)

	if got := user.GetQuota(db); math.Abs(float64(got)) > 0.001 {
		t.Fatalf("expected quota to be drained to 0, got %f", got)
	}
	if got := user.GetUsedQuota(db); math.Abs(float64(got-1)) > 0.001 {
		t.Fatalf("expected used quota to record only paid balance 1, got %f", got)
	}
}

func TestRealtimeQuotaLimiterRejectsProjectedSubscriptionOverflow(t *testing.T) {
	useChatTestChargeInstance(t)

	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 10) {
		t.Fatalf("set quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, false) {
		t.Fatalf("disable subscription quota fallback")
	}

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: 1,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	limiter := newRealtimeQuotaLimiter(db, cache, user, globals.GPT3Turbo, true)
	if limiter.allowsProjectedChunk(buffer, &globals.Chunk{Content: "hello"}) {
		t.Fatalf("expected realtime limiter to reject a chunk above subscription budget")
	}
	if !buffer.IsEmpty() {
		t.Fatalf("expected rejected projected chunk not to mutate buffer")
	}
}

func TestRealtimeQuotaLimiterAllowsSubscriptionOverflowWhenFallbackEnabled(t *testing.T) {
	useChatTestChargeInstance(t)

	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 10) {
		t.Fatalf("set quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, true) {
		t.Fatalf("enable subscription quota fallback")
	}

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: 1,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	limiter := newRealtimeQuotaLimiter(db, cache, user, globals.GPT3Turbo, true)
	if !limiter.allowsProjectedChunk(buffer, &globals.Chunk{Content: "hello"}) {
		t.Fatalf("expected realtime limiter to allow overflow covered by user quota")
	}
	if !buffer.IsEmpty() {
		t.Fatalf("expected projected chunk not to mutate buffer")
	}
}

func TestRealtimeQuotaLimiterRejectsOfficialUsageOverflow(t *testing.T) {
	useChatTokenTestChargeInstance(t)

	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 10) {
		t.Fatalf("set quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, false) {
		t.Fatalf("disable subscription quota fallback")
	}

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: 0.49,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTokenTestCharge{})
	limiter := newRealtimeQuotaLimiter(db, cache, user, globals.GPT3Turbo, true)
	if limiter.allowsProjectedChunk(buffer, &globals.Chunk{
		Usage: &globals.TokenUsage{
			CompletionTokens: 720,
			TotalTokens:      720,
		},
	}) {
		t.Fatalf("expected realtime limiter to reject official usage above subscription budget")
	}
	if !buffer.IsEmpty() {
		t.Fatalf("expected rejected projected usage not to mutate buffer")
	}
}

func TestRealtimeQuotaLimiterRejectsSplitBufferOfficialUsageOverflow(t *testing.T) {
	liveBuffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTokenTestCharge{})
	roundBuffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTokenTestCharge{})
	limiter := realtimeQuotaLimiter{enabled: true, limit: 0.49}

	chunk := &globals.Chunk{
		Usage: &globals.TokenUsage{
			CompletionTokens: 720,
			TotalTokens:      720,
		},
	}

	if limiter.allowsProjectedSplitChunk(liveBuffer, roundBuffer, chunk) {
		t.Fatalf("expected split-buffer limiter to reject official usage above budget")
	}
	if !liveBuffer.IsEmpty() || !roundBuffer.IsEmpty() {
		t.Fatalf("expected split-buffer projection not to mutate buffers")
	}
}

func TestToolRoundUsageMergesIntoLiveBilling(t *testing.T) {
	liveBuffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTokenTestCharge{})
	roundBuffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTokenTestCharge{})
	roundBuffer.WriteChunk(&globals.Chunk{
		Usage: &globals.TokenUsage{
			PromptTokens:     30,
			CompletionTokens: 720,
			TotalTokens:      750,
		},
	})

	syncToolFinalMetadata(liveBuffer, roundBuffer)

	if got := liveBuffer.GetRecordQuota(); math.Abs(float64(got-0.72)) > 0.001 {
		t.Fatalf("expected live billing to include tool round official usage, got %f", got)
	}
}

func TestCollectQuotaChargesUserBalanceForSubscriptionOverflow(t *testing.T) {
	useChatTestChargeInstance(t)

	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 10) {
		t.Fatalf("set quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, true) {
		t.Fatalf("enable subscription quota fallback")
	}

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: 1,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	buffer.Write("hello")

	CollectQuota(newCollectQuotaTestContextWithCache(t, db, cache), user, buffer, true, nil)

	plan := channel.PlanInstance.GetPlan(1)
	if got := plan.GetPointUsage(user, cache); math.Abs(float64(got-1)) > 0.001 {
		t.Fatalf("expected subscription usage to consume available 1 quota, got %f", got)
	}
	if got := user.GetQuota(db); math.Abs(float64(got-2)) > 0.001 {
		t.Fatalf("expected user quota to pay subscription overflow, got %f", got)
	}
	if got := user.GetUsedQuota(db); math.Abs(float64(got-8)) > 0.001 {
		t.Fatalf("expected used quota to record subscription overflow, got %f", got)
	}
}

func TestCollectQuotaCapsSubscriptionFallbackAtUserBalance(t *testing.T) {
	useChatTestChargeInstance(t)

	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 3) {
		t.Fatalf("set quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, true) {
		t.Fatalf("enable subscription quota fallback")
	}

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: 1,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	buffer.Write("hello")

	CollectQuota(newCollectQuotaTestContextWithCache(t, db, cache), user, buffer, true, nil)

	plan := channel.PlanInstance.GetPlan(1)
	if got := plan.GetPointUsage(user, cache); math.Abs(float64(got-1)) > 0.001 {
		t.Fatalf("expected subscription usage to consume available 1 quota, got %f", got)
	}
	if got := user.GetQuota(db); math.Abs(float64(got)) > 0.001 {
		t.Fatalf("expected user quota to be drained to 0, got %f", got)
	}
	if got := user.GetUsedQuota(db); math.Abs(float64(got-3)) > 0.001 {
		t.Fatalf("expected used quota to record only available balance 3, got %f", got)
	}
}

func TestCollectQuotaDoesNotChargeUserBalanceWhenSubscriptionFallbackDisabled(t *testing.T) {
	useChatTestChargeInstance(t)

	db := openChatQuotaTestDB(t)
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}
	if !user.SetQuota(db, 10) {
		t.Fatalf("set quota")
	}
	if !user.SetAllowSubscriptionQuotaFallback(db, false) {
		t.Fatalf("disable subscription quota fallback")
	}

	previousPlan := channel.PlanInstance
	channel.PlanInstance = &channel.PlanManager{
		Enabled: true,
		Plans: []channel.Plan{
			{
				Level: 1,
				Quota: 1,
				Items: []channel.PlanItem{
					{Id: "included", Models: []string{globals.GPT3Turbo}},
				},
			},
		},
	}
	t.Cleanup(func() {
		channel.PlanInstance = previousPlan
	})

	if _, err := globals.ExecDb(
		db,
		"INSERT INTO subscription (user_id, level, expired_at) VALUES (?, ?, ?)",
		user.GetID(db),
		1,
		time.Now().Add(24*time.Hour).Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	buffer := utils.NewBuffer(globals.GPT3Turbo, nil, chatTestCharge{})
	buffer.Write("hello")

	CollectQuota(newCollectQuotaTestContextWithCache(t, db, cache), user, buffer, true, nil)

	plan := channel.PlanInstance.GetPlan(1)
	if got := plan.GetPointUsage(user, cache); math.Abs(float64(got-1)) > 0.001 {
		t.Fatalf("expected subscription usage to consume available 1 quota, got %f", got)
	}
	if got := user.GetQuota(db); math.Abs(float64(got-10)) > 0.001 {
		t.Fatalf("expected user quota not to pay subscription overflow, got %f", got)
	}
	if got := user.GetUsedQuota(db); math.Abs(float64(got)) > 0.001 {
		t.Fatalf("expected used quota to remain 0, got %f", got)
	}
}

func TestCreateStopSignalEmitsStopAndCancelsPolling(t *testing.T) {
	conn := NewConnection(nil, false, "", 2)
	conn.Write(&conversation.FormMessage{Type: StopType})

	stopSignal, cancel := createStopSignal(conn, nil)
	defer cancel()

	select {
	case stopped := <-stopSignal:
		if !stopped {
			t.Fatalf("expected stop signal")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for stop signal")
	}

	cancel()
}

func TestCreateStopSignalHandlesRemoveWithoutStopping(t *testing.T) {
	conn := NewConnection(nil, false, "", 3)
	conn.Write(&conversation.FormMessage{Type: RemoveType, Message: "2"})

	removed := make(chan int, 1)
	stopSignal, cancel := createStopSignal(conn, func(index int) {
		removed <- index
	})
	defer cancel()

	select {
	case index := <-removed:
		if index != 2 {
			t.Fatalf("expected remove index 2, got %d", index)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for remove handler")
	}

	select {
	case stopped := <-stopSignal:
		if stopped {
			t.Fatalf("remove event should not stop chat request")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for non-stop signal")
	}
}

func TestCreateStopSignalConsumesStopAfterRemove(t *testing.T) {
	conn := NewConnection(nil, false, "", 3)
	conn.Write(&conversation.FormMessage{Type: RemoveType, Message: "1"})
	conn.Write(&conversation.FormMessage{Type: StopType})

	removed := make(chan int, 1)
	stopSignal, cancel := createStopSignal(conn, func(index int) {
		removed <- index
	})
	defer cancel()

	select {
	case index := <-removed:
		if index != 1 {
			t.Fatalf("expected remove index 1, got %d", index)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for remove handler")
	}

	select {
	case stopped := <-stopSignal:
		if !stopped {
			t.Fatalf("expected stop signal after queued remove event")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for stop signal")
	}
}
