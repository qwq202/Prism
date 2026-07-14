package manager

import (
	"chat/auth"
	"chat/channel"
	"chat/connection"
	"chat/globals"
	"chat/manager/askuser"
	"chat/manager/conversation"
	"chat/middleware"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

const websocketTestModel = "ws-test-model"

func openWebsocketTestDB(t *testing.T) *sql.DB {
	t.Helper()

	previous := globals.SqliteEngine
	globals.SqliteEngine = true
	t.Cleanup(func() {
		globals.SqliteEngine = previous
	})

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "chat-websocket-e2e.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	connection.CreateUserTable(db)
	connection.CreateConversationTable(db)
	connection.CreateChatRequestTable(db)
	connection.CreateQuotaTable(db)
	connection.CreateSubscriptionTable(db)
	connection.CreateBillingTable(db)

	return db
}

func openWebsocketTestCache(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	cache := redis.NewClient(&redis.Options{Addr: server.Addr()})
	if err := cache.Ping(cache.Context()).Err(); err != nil {
		t.Fatalf("ping miniredis: %v", err)
	}

	t.Cleanup(func() {
		_ = cache.Close()
		server.Close()
	})

	return server, cache
}

func rootToken(t *testing.T, db *sql.DB) (int64, string) {
	t.Helper()

	user := auth.GetUserByName(db, "root")
	if user == nil {
		t.Fatalf("expected root user")
	}

	token, err := user.GenerateTokenSafe(db)
	if err != nil {
		t.Fatalf("generate root token: %v", err)
	}

	return user.GetID(db), token
}

func newSlowStreamingUpstream() (*httptest.Server, *int32) {
	var requests int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		writeChunk := func(chunk string) {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}

		writeChunk(`{"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"ws-test-model","choices":[{"index":0,"delta":{"content":"first chunk"},"finish_reason":""}]}`)
		time.Sleep(350 * time.Millisecond)
		writeChunk(`{"id":"chatcmpl-test","object":"chat.completion.chunk","created":1,"model":"ws-test-model","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup_weather","arguments":"{\"city\":\"Shanghai\"}"}}]},"finish_reason":""}]}`)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))

	return server, &requests
}

func newAskUserUpstream() (*httptest.Server, *int32) {
	var requests int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestNumber := atomic.AddInt32(&requests, 1)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		if requestNumber == 1 {
			arguments := `{"questions":[{"id":"scope","header":"Scope","question":"Which scope should be implemented?","type":"single","options":[{"label":"Minimal","description":"Only the core flow."},{"label":"Complete","description":"Include recovery and tests."}]},{"id":"parts","header":"Parts","question":"Which parts should be included?","type":"multiple","options":[{"label":"UI","description":"Build the answer card."},{"label":"Tests","description":"Cover the continuation flow."}]}]}`
			chunk := fmt.Sprintf(
				`{"id":"chatcmpl-ask","object":"chat.completion.chunk","created":1,"model":"ws-test-model","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_ask_1","type":"function","function":{"name":"ask_user","arguments":%q}}]},"finish_reason":"tool_calls"}]}`,
				arguments,
			)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
		} else {
			_, _ = fmt.Fprint(w, `data: {"id":"chatcmpl-final","object":"chat.completion.chunk","created":1,"model":"ws-test-model","choices":[{"index":0,"delta":{"content":"Continuing with the complete UI and tests."},"finish_reason":"stop"}]}`+"\n\n")
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))

	return server, &requests
}

func readWebsocketResponse(t *testing.T, conn *websocket.Conn) globals.ChatSegmentResponse {
	t.Helper()

	for {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		var response globals.ChatSegmentResponse
		if err := conn.ReadJSON(&response); err != nil {
			t.Fatalf("read websocket response: %v", err)
		}
		if response.ResponseType == "capabilities" {
			continue
		}

		return response
	}
}

func waitForConversationMessages(t *testing.T, db *sql.DB, userID int64, conversationID int64, count int) *conversation.Conversation {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		instance := conversation.LoadConversation(db, userID, conversationID)
		if instance != nil && instance.GetMessageLength() == count {
			return instance
		}

		time.Sleep(25 * time.Millisecond)
	}

	instance := conversation.LoadConversation(db, userID, conversationID)
	if instance == nil {
		t.Fatalf("conversation %d was not persisted", conversationID)
	}

	t.Fatalf("expected conversation %d to have %d messages, got %d", conversationID, count, instance.GetMessageLength())
	return nil
}

type websocketChatTestEnv struct {
	db           *sql.DB
	rootID       int64
	token        string
	server       *httptest.Server
	requestCount *int32
}

func newWebsocketChatTestEnv(t *testing.T) *websocketChatTestEnv {
	t.Helper()
	upstream, requestCount := newSlowStreamingUpstream()
	return newWebsocketChatTestEnvWithUpstream(t, upstream, requestCount)
}

func newWebsocketChatTestEnvWithUpstream(t *testing.T, upstream *httptest.Server, requestCount *int32) *websocketChatTestEnv {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db := openWebsocketTestDB(t)
	_, cache := openWebsocketTestCache(t)
	rootID, token := rootToken(t, db)

	t.Cleanup(upstream.Close)

	previousConduit := channel.ConduitInstance
	previousCharge := channel.ChargeInstance
	previousPlan := channel.PlanInstance
	previousConnectionCache := connection.Cache
	previousConnectionDB := connection.DB
	previousTaskModel := globals.GetTaskModel()
	previousCacheModels := globals.CacheAcceptedModels
	previousCacheSize := globals.CacheAcceptedSize

	channel.ConduitInstance = &channel.Manager{
		Sequence: channel.Sequence{
			&channel.Channel{
				Id:       1,
				Name:     "websocket-test",
				Type:     globals.OpenAIChannelType,
				Endpoint: upstream.URL,
				Models:   []string{websocketTestModel},
				State:    true,
			},
		},
		PreflightSequence: map[string]channel.Sequence{},
	}
	channel.ConduitInstance.Load()

	channel.ChargeInstance = &channel.ChargeManager{
		Models:           map[string]*channel.Charge{},
		NonBillingModels: []string{},
	}
	channel.PlanInstance = &channel.PlanManager{}

	connection.Cache = cache
	connection.DB = db
	globals.SetTaskModel("")
	globals.CacheAcceptedModels = nil
	globals.CacheAcceptedSize = 1

	t.Cleanup(func() {
		channel.ConduitInstance = previousConduit
		channel.ChargeInstance = previousCharge
		channel.PlanInstance = previousPlan
		connection.Cache = previousConnectionCache
		connection.DB = previousConnectionDB
		globals.SetTaskModel(previousTaskModel)
		globals.CacheAcceptedModels = previousCacheModels
		globals.CacheAcceptedSize = previousCacheSize
	})

	router := gin.New()
	router.Use(middleware.BuiltinMiddleWare(db, cache))
	managerGroup := router.Group("")
	Register(managerGroup)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &websocketChatTestEnv{
		db:           db,
		rootID:       rootID,
		token:        token,
		server:       server,
		requestCount: requestCount,
	}
}

func TestChatAPIWebsocketAskUserAnswerContinuesGeneration(t *testing.T) {
	upstream, requestCount := newAskUserUpstream()
	env := newWebsocketChatTestEnvWithUpstream(t, upstream, requestCount)

	conn := env.dial(t, -1)
	defer conn.Close()

	if err := conn.WriteJSON(conversation.FormMessage{
		Type:    ChatType,
		Message: "Build the feature, but ask me when a decision is required.",
		Model:   websocketTestModel,
	}); err != nil {
		t.Fatalf("send chat message: %v", err)
	}

	var conversationID int64
	var pendingEvent *globals.ChatSegmentToolCall
	for {
		response := readWebsocketResponse(t, conn)
		if response.Conversation != 0 {
			conversationID = response.Conversation
		}
		if response.ToolCall != nil && response.ToolCall.Name == askuser.ToolName {
			pendingEvent = response.ToolCall
		}
		if response.End {
			break
		}
	}
	if conversationID == 0 {
		t.Fatalf("expected conversation promotion")
	}
	if pendingEvent == nil || pendingEvent.Status != "pending" || pendingEvent.Id != "call_ask_1" {
		t.Fatalf("unexpected pending ask_user event: %#v", pendingEvent)
	}

	pending := waitForConversationMessages(t, env.db, env.rootID, conversationID, 2)
	pendingAssistant := pending.GetMessage()[1]
	if pendingAssistant.ToolCalls == nil || len(*pendingAssistant.ToolCalls) != 1 {
		t.Fatalf("expected pending tool call to be persisted: %#v", pendingAssistant)
	}
	if _, _, err := askuser.NormalizeToolCall((*pendingAssistant.ToolCalls)[0]); err != nil {
		t.Fatalf("expected persisted tool arguments to be normalized: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close pending websocket: %v", err)
	}
	conn = env.dial(t, conversationID)
	defer conn.Close()

	answer := json.RawMessage(`{
		"type":"ask_user_answer",
		"answers":{
			"scope":{"type":"single","value":"Complete","custom":false,"skipped":false},
			"parts":{"type":"multiple","value":["UI","Tests"],"custom":false,"skipped":false}
		}
	}`)
	if err := conn.WriteJSON(conversation.FormMessage{
		Type:       ToolResultType,
		Model:      websocketTestModel,
		ToolCallID: "call_ask_1",
		ToolResult: answer,
	}); err != nil {
		t.Fatalf("send ask_user answer: %v", err)
	}

	var continued string
	for {
		response := readWebsocketResponse(t, conn)
		continued += response.Message
		if response.End {
			break
		}
	}
	if continued != "Continuing with the complete UI and tests." {
		t.Fatalf("unexpected continued response %q", continued)
	}

	completed := waitForConversationMessages(t, env.db, env.rootID, conversationID, 4)
	messages := completed.GetMessage()
	if messages[2].Role != globals.Tool || messages[2].ToolCallId == nil || *messages[2].ToolCallId != "call_ask_1" {
		t.Fatalf("expected persisted ask_user tool result, got %#v", messages[2])
	}
	if messages[3].Role != globals.Assistant || messages[3].Content != continued {
		t.Fatalf("expected continued assistant response, got %#v", messages[3])
	}
	if got := atomic.LoadInt32(requestCount); got != 2 {
		t.Fatalf("expected one question request and one continuation request, got %d", got)
	}
}

func (e *websocketChatTestEnv) dial(t *testing.T, conversationID int64) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(e.server.URL, "http") + "/chat"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	if err := conn.WriteJSON(WebsocketAuthForm{
		Token: e.token,
		Id:    conversationID,
	}); err != nil {
		t.Fatalf("send websocket auth: %v", err)
	}

	return conn
}

func TestChatAPIWebsocketStopAndRestartPersistedHistory(t *testing.T) {
	env := newWebsocketChatTestEnv(t)

	conn := env.dial(t, -1)
	defer conn.Close()

	if err := conn.WriteJSON(conversation.FormMessage{
		Type:    ChatType,
		Message: "hello from websocket",
		Model:   websocketTestModel,
	}); err != nil {
		t.Fatalf("send chat message: %v", err)
	}

	var conversationID int64
	sentStop := false

	for {
		response := readWebsocketResponse(t, conn)
		if response.Conversation != 0 && conversationID == 0 {
			conversationID = response.Conversation
			continue
		}

		if response.Message == "first chunk" && !sentStop {
			sentStop = true
			if err := conn.WriteJSON(conversation.FormMessage{Type: StopType}); err != nil {
				t.Fatalf("send stop message: %v", err)
			}
		}

		if response.End {
			break
		}
	}

	if conversationID == 0 {
		t.Fatalf("expected server to assign a conversation id")
	}

	afterStop := waitForConversationMessages(t, env.db, env.rootID, conversationID, 2)
	if afterStop.GetMessage()[0].Role != globals.User || afterStop.GetMessage()[0].Content != "hello from websocket" {
		t.Fatalf("unexpected persisted user message after stop: %#v", afterStop.GetMessage()[0])
	}

	stoppedAssistant := afterStop.GetMessage()[1]
	if stoppedAssistant.Role != globals.Assistant {
		t.Fatalf("expected assistant response after stop, got %#v", stoppedAssistant)
	}
	if stoppedAssistant.Content != "first chunk" {
		t.Fatalf("expected interrupted assistant text to persist, got %q", stoppedAssistant.Content)
	}
	if stoppedAssistant.ToolCalls != nil || stoppedAssistant.FunctionCall != nil {
		t.Fatalf("expected interrupted assistant payloads to be dropped, got tool_calls=%#v function_call=%#v", stoppedAssistant.ToolCalls, stoppedAssistant.FunctionCall)
	}

	if err := conn.WriteJSON(conversation.FormMessage{
		Type:  RestartType,
		Model: websocketTestModel,
	}); err != nil {
		t.Fatalf("send restart message: %v", err)
	}

	for {
		response := readWebsocketResponse(t, conn)
		if response.End {
			break
		}
	}

	afterRestart := waitForConversationMessages(t, env.db, env.rootID, conversationID, 3)
	restartedAssistant := afterRestart.GetMessage()[2]
	if restartedAssistant.Role != globals.Assistant {
		t.Fatalf("expected restarted assistant response, got %#v", restartedAssistant)
	}
	if restartedAssistant.Content != "first chunk" {
		t.Fatalf("expected restarted assistant text to persist, got %q", restartedAssistant.Content)
	}
	if restartedAssistant.ToolCalls == nil || len(*restartedAssistant.ToolCalls) != 1 {
		t.Fatalf("expected restart to persist tool payloads, got %#v", restartedAssistant.ToolCalls)
	}

	call := (*restartedAssistant.ToolCalls)[0]
	if call.Function.Name != "lookup_weather" || call.Function.Arguments != "{\"city\":\"Shanghai\"}" {
		t.Fatalf("unexpected restarted tool payload: %#v", call)
	}

	if got := atomic.LoadInt32(env.requestCount); got != 2 {
		t.Fatalf("expected stop + restart to issue exactly two upstream requests without retrying the interruption, got %d", got)
	}
}

func TestChatAPIWebsocketTransientChatSkipsPersistence(t *testing.T) {
	env := newWebsocketChatTestEnv(t)

	conn := env.dial(t, -1)
	defer conn.Close()

	if err := conn.WriteJSON(conversation.FormMessage{
		Type:      ChatType,
		Message:   "draw a small pig",
		Model:     websocketTestModel,
		Transient: true,
	}); err != nil {
		t.Fatalf("send transient chat message: %v", err)
	}

	var streamed string
	for {
		response := readWebsocketResponse(t, conn)
		if response.Conversation != 0 {
			t.Fatalf("transient request should not promote a conversation id, got %d", response.Conversation)
		}

		streamed += response.Message
		if response.End {
			break
		}
	}

	if !strings.Contains(streamed, "first chunk") {
		t.Fatalf("expected transient request to stream upstream response, got %q", streamed)
	}
	if persisted := conversation.LoadConversation(env.db, env.rootID, 1); persisted != nil {
		t.Fatalf("expected transient conversation to skip persistence, got %#v", persisted.GetMessage())
	}
	if got := atomic.LoadInt32(env.requestCount); got != 1 {
		t.Fatalf("expected transient request to issue one upstream request, got %d", got)
	}
}

func TestChatAPIWebsocketCloseContinuesAndPersistsFullResponse(t *testing.T) {
	env := newWebsocketChatTestEnv(t)

	conn := env.dial(t, -1)

	if err := conn.WriteJSON(conversation.FormMessage{
		Type:    ChatType,
		Message: "close while streaming",
		Model:   websocketTestModel,
	}); err != nil {
		t.Fatalf("send chat message: %v", err)
	}

	var conversationID int64
	for {
		response := readWebsocketResponse(t, conn)
		if response.Conversation != 0 {
			conversationID = response.Conversation
			continue
		}

		if response.Message == "first chunk" {
			break
		}
	}

	if conversationID == 0 {
		t.Fatalf("expected server to assign a conversation id")
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket: %v", err)
	}

	afterClose := waitForConversationMessages(t, env.db, env.rootID, conversationID, 2)
	if afterClose.GetMessage()[0].Role != globals.User || afterClose.GetMessage()[0].Content != "close while streaming" {
		t.Fatalf("unexpected persisted user message after close: %#v", afterClose.GetMessage()[0])
	}

	assistant := afterClose.GetMessage()[1]
	if assistant.Role != globals.Assistant {
		t.Fatalf("expected assistant response after close, got %#v", assistant)
	}
	if assistant.Content != "first chunk" {
		t.Fatalf("expected streamed assistant text to persist, got %q", assistant.Content)
	}
	if assistant.ToolCalls == nil || len(*assistant.ToolCalls) != 1 {
		t.Fatalf("expected close to keep full assistant payloads, got %#v", assistant.ToolCalls)
	}

	call := (*assistant.ToolCalls)[0]
	if call.Function.Name != "lookup_weather" || call.Function.Arguments != "{\"city\":\"Shanghai\"}" {
		t.Fatalf("unexpected assistant tool payload after close: %#v", call)
	}

	if got := atomic.LoadInt32(env.requestCount); got != 1 {
		t.Fatalf("expected close to keep the original upstream request only, got %d", got)
	}
}

func TestChatAPIWebsocketDuplicateRequestIDDoesNotRegenerate(t *testing.T) {
	env := newWebsocketChatTestEnv(t)

	const requestID = "ws-request-once"
	conn := env.dial(t, -1)
	if err := conn.WriteJSON(conversation.FormMessage{
		Type:      ChatType,
		Message:   "send this once",
		Model:     websocketTestModel,
		RequestID: requestID,
	}); err != nil {
		t.Fatalf("send chat message: %v", err)
	}

	var conversationID int64
	var accepted bool
	for {
		response := readWebsocketResponse(t, conn)
		if response.RequestID == requestID && response.Accepted {
			accepted = true
		}
		if response.Conversation != 0 {
			conversationID = response.Conversation
		}
		if response.RequestID == requestID && response.RequestStatus == conversation.ChatRequestCompleted {
			break
		}
	}
	if !accepted || conversationID == 0 {
		t.Fatalf("expected durable request acknowledgement, accepted=%v conversation=%d", accepted, conversationID)
	}
	_ = conn.Close()

	persisted := waitForConversationMessages(t, env.db, env.rootID, conversationID, 2)
	if persisted.GetMessage()[0].RequestID != requestID {
		t.Fatalf("expected request id on persisted user message, got %#v", persisted.GetMessage()[0])
	}
	if persisted.GetMessage()[1].RequestID != requestID {
		t.Fatalf("expected request id on persisted assistant message, got %#v", persisted.GetMessage()[1])
	}

	replay := env.dial(t, conversationID)
	defer replay.Close()
	if err := replay.WriteJSON(conversation.FormMessage{
		Type:      ChatType,
		Message:   "send this once",
		Model:     websocketTestModel,
		RequestID: requestID,
	}); err != nil {
		t.Fatalf("replay chat message: %v", err)
	}

	response := readWebsocketResponse(t, replay)
	if response.RequestID != requestID || response.RequestStatus != conversation.ChatRequestCompleted || !response.Accepted || response.End {
		t.Fatalf("unexpected replay acknowledgement: %#v", response)
	}
	if got := atomic.LoadInt32(env.requestCount); got != 1 {
		t.Fatalf("expected replay to avoid a second upstream request, got %d", got)
	}
	if got := waitForConversationMessages(t, env.db, env.rootID, conversationID, 2).GetMessageLength(); got != 2 {
		t.Fatalf("expected replay to preserve exactly two messages, got %d", got)
	}
}

func TestChatAPIWebsocketDeclaresRequestAckCapability(t *testing.T) {
	env := newWebsocketChatTestEnv(t)
	conn := env.dial(t, -1)
	defer conn.Close()
	if err := conn.WriteJSON(conversation.FormMessage{Type: CapabilitiesType}); err != nil {
		t.Fatalf("request capability frame: %v", err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var response globals.ChatSegmentResponse
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("read capability frame: %v", err)
	}
	if response.ResponseType != "capabilities" {
		t.Fatalf("expected capabilities frame, got %#v", response)
	}
	found := false
	for _, capability := range response.Capabilities {
		if capability == chatRequestAckCapability {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected request ACK capability, got %#v", response.Capabilities)
	}
}
