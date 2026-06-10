package manager

import (
	adaptercommon "chat/adapter/common"
	"chat/addition/web"
	"chat/admin"
	"chat/auth"
	"chat/billing"
	"chat/channel"
	"chat/globals"
	"chat/manager/memory"
	"chat/utils"
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func NativeChatHandler(c *gin.Context, user *auth.User, model string, message []globals.Message, enableWeb bool) (string, float32) {
	defer func() {
		if err := recover(); err != nil {
			stack := debug.Stack()
			globals.Warn(fmt.Sprintf("caught panic from chat handler: %s (instance: %s, client: %s)\n%s",
				err, model, c.ClientIP(), stack,
			))
		}
	}()

	db := utils.GetDBFromContext(c)
	cache := utils.GetCacheFromContext(c)
	group := auth.GetGroup(db, user)
	toolCallsSupported := memory.CanUseToolCalls(model, group)
	segment := utils.DeepCopy(message)
	if web.ShouldUseFallbackSearch(enableWeb, model, toolCallsSupported) {
		segment = web.ToFallbackSearched(segment, group, cache)
	}
	check, plan := auth.CanEnableModelWithSubscription(db, cache, user, model, segment)

	if check != nil {
		return check.Error(), 0
	}

	buffer := utils.NewBuffer(model, segment, channel.ChargeInstance.GetCharge(model))
	limiter := newRealtimeQuotaLimiter(db, cache, user, model, plan)
	buildProps := func(
		segment []globals.Message,
		requestBuffer *utils.Buffer,
		tools *globals.FunctionTools,
		toolChoice *interface{},
		disableCache bool,
	) *adaptercommon.ChatProps {
		props := adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
			Model:            model,
			OriginalModel:    model,
			Message:          segment,
			Tools:            tools,
			ToolChoice:       toolChoice,
			EnableWeb:        enableWeb,
			EnableWebSearch:  enableWeb,
			EnableURLContext: enableWeb,
			EnableXSearch:    false,
			ClientContext:    extractClientContext(c),
			DisableCache:     disableCache,
		}, requestBuffer)
		limiter.applyMaxOutputTokens(props, buffer)
		return props
	}

	var hit bool
	var err error
	webSearchToolEnabled := canUseTavilySearchTool(enableWeb, model, toolCallsSupported)
	tools := buildAvailableToolDefinitions(false, false, webSearchToolEnabled)
	if tools != nil {
		hit, err = createNativeToolChatTask(buffer, model, group, segment, tools, memory.MaxToolRounds, buildProps)
	} else {
		hit, err = channel.NewChatRequestWithCache(
			cache, buffer,
			group,
			buildProps(segment, buffer, nil, nil, false),
			func(resp *globals.Chunk) error {
				buffer.WriteChunk(resp)
				return nil
			},
		)
	}

	admin.AnalyseRequest(model, buffer, err)
	billing.RecordModelUsageMetric(db, model, buffer, err)
	if err != nil {
		auth.RevertSubscriptionUsage(db, cache, user, model)
		return err.Error(), 0
	}

	if !hit {
		CollectQuota(c, user, buffer, plan, err)
		createChatBillingRecord(db, user, model, buffer)
	}

	return buffer.ReadWithDefault(defaultMessage), buffer.GetRecordQuota()
}
