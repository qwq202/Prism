package drawing

import (
	adaptercommon "chat/adapter/common"
	"chat/admin"
	"chat/auth"
	"chat/billing"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const drawingTaskCanceledMessage = "drawing task canceled"

var errDrawingTaskCanceled = errors.New(drawingTaskCanceledMessage)

func StartTask(db *sql.DB, cache *redis.Client, userID int64, taskID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				globals.Warn(fmt.Sprintf("[drawing] task panic: %s (task: %s)\n%s", r, taskID, stack))
				_ = MarkTaskFailed(db, taskID, fmt.Errorf("%v", r))
			}
		}()

		runTask(db, cache, userID, taskID)
	}()
}

func runTask(db *sql.DB, cache *redis.Client, userID int64, taskID string) {
	if IsTaskCanceled(db, taskID) {
		return
	}
	if err := MarkTaskRunning(db, taskID); err != nil {
		globals.Warn(fmt.Sprintf("[drawing] failed to mark task running: %s (task: %s)", err.Error(), taskID))
		return
	}

	task, err := LoadTask(db, userID, taskID)
	if err != nil {
		globals.Warn(fmt.Sprintf("[drawing] failed to load task: %s (task: %s)", err.Error(), taskID))
		return
	}
	if task.Status == TaskStatusCanceled {
		return
	}

	user := auth.GetUserById(db, userID)
	if user == nil {
		_ = MarkTaskFailed(db, taskID, fmt.Errorf("user not found"))
		return
	}

	images, quota, err := executeTaskRequest(db, cache, user, task)
	if errors.Is(err, errDrawingTaskCanceled) || IsTaskCanceled(db, taskID) {
		return
	}
	if err != nil {
		_ = MarkTaskFailed(db, taskID, err)
		return
	}
	if len(images) == 0 {
		_ = MarkTaskFailed(db, taskID, fmt.Errorf("no image generated"))
		return
	}

	if IsTaskCanceled(db, taskID) {
		return
	}
	if err := AppendImagesToWorkspace(db, userID, task.WorkspaceID, task.Model, images, task.Prompt); err != nil {
		_ = MarkTaskFailed(db, taskID, err)
		return
	}
	if IsTaskCanceled(db, taskID) {
		return
	}
	_ = MarkTaskSucceeded(db, taskID, images, quota)
}

func executeTaskRequest(db *sql.DB, cache *redis.Client, user *auth.User, task *Task) ([]GeneratedImage, float64, error) {
	if task == nil {
		return nil, 0, fmt.Errorf("task is nil")
	}

	responseFormat := rawJSONToInterface(task.Options.ResponseFormat)
	thinking := rawJSONToInterface(task.Options.Thinking)
	messages := []globals.Message{
		{
			Role:    globals.User,
			Content: task.Message,
		},
	}

	check, plan := auth.CanEnableModelWithSubscriptionForRequest(db, cache, user, task.Model, messages, responseFormat)
	if check != nil {
		return nil, 0, check
	}

	buffer := utils.NewBuffer(task.Model, messages, channel.ChargeInstance.GetCharge(task.Model))
	group := auth.GetGroup(db, user)
	props := adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
		Model:          task.Model,
		OriginalModel:  task.Model,
		Message:        messages,
		ResponseFormat: responseFormat,
		Thinking:       thinking,
		DisableCache:   false,
	}, buffer)

	hit, err := channel.NewChatRequestWithCache(cache, buffer, group, props, func(data *globals.Chunk) error {
		if IsTaskCanceled(db, task.TaskID) {
			return errDrawingTaskCanceled
		}
		buffer.WriteChunk(data)
		return nil
	})

	admin.AnalyseRequest(task.Model, buffer, err)
	billing.RecordModelUsageMetric(db, task.Model, buffer, err)
	if errors.Is(err, errDrawingTaskCanceled) {
		if buffer.IsEmpty() {
			auth.RevertSubscriptionUsage(db, cache, user, task.Model)
		}
		return nil, 0, err
	}
	if err != nil {
		if buffer.HasVisiblePayload() {
			collectTaskQuota(db, cache, user, buffer, plan, err)
			createTaskBillingRecord(db, user, task.Model, buffer)
		} else {
			auth.RevertSubscriptionUsage(db, cache, user, task.Model)
		}
		return nil, 0, err
	}

	if !hit {
		collectTaskQuota(db, cache, user, buffer, plan, nil)
		createTaskBillingRecord(db, user, task.Model, buffer)
	}

	content := buffer.Read()
	images, err := generatedImagesFromContent(content, task.Prompt, task.TaskID)
	if err != nil {
		return nil, float64(buffer.GetRecordQuota()), err
	}
	if len(images) == 0 && strings.TrimSpace(content) != "" {
		return nil, float64(buffer.GetRecordQuota()), errors.New(strings.TrimSpace(content))
	}
	return images, float64(buffer.GetRecordQuota()), nil
}

func rawJSONToInterface(raw json.RawMessage) interface{} {
	raw = normalizeJSONRaw(raw)
	if len(raw) == 0 {
		return nil
	}

	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func generatedImagesFromContent(content string, prompt string, taskID string) ([]GeneratedImage, error) {
	_, sources := utils.ExtractImages(content, true)
	if len(sources) == 0 {
		return nil, nil
	}

	createdAt := time.Now().UnixMilli()
	images := make([]GeneratedImage, 0, len(sources))
	for index, source := range sources {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		stored, _, err := storeDrawingDataURL(source)
		if err != nil {
			return nil, err
		}
		images = append(images, GeneratedImage{
			ID:        fmt.Sprintf("%s-%d", taskID, index),
			Src:       stored,
			Prompt:    prompt,
			CreatedAt: createdAt,
		})
	}
	return images, nil
}

func collectTaskQuota(db *sql.DB, cache *redis.Client, user *auth.User, buffer *utils.Buffer, plan bool, err error) {
	if user == nil || buffer == nil {
		return
	}

	quota := buffer.GetRecordQuota()
	if quota <= 0 || buffer.IsEmpty() {
		return
	}
	if err != nil {
		globals.Warn(fmt.Sprintf("[drawing] charging visible partial response after request error (model: %s): %s", buffer.GetModel(), err.Error()))
	}

	if plan {
		consumed := auth.FinalizeSubscriptionUsageAmount(db, cache, user, buffer.GetModel(), quota)
		if consumed+0.0001 < quota {
			if user.AllowSubscriptionQuotaFallback(db) {
				collectTaskUserQuota(db, user, quota-consumed)
			} else {
				globals.Warn(fmt.Sprintf(
					"[drawing] subscription usage only covered %.4f/%.4f quota and credit fallback is disabled (model: %s)",
					consumed,
					quota,
					buffer.GetModel(),
				))
			}
		}
		return
	}

	collectTaskUserQuota(db, user, quota)
}

func collectTaskUserQuota(db *sql.DB, user *auth.User, quota float32) {
	if !user.ChargeQuota(db, quota) {
		globals.Warn(fmt.Sprintf(
			"[drawing] user quota only partially covered %.4f quota; balance has been drained without creating debt (user: %s)",
			quota,
			user.Username,
		))
	}
}

func createTaskBillingRecord(db *sql.DB, user *auth.User, model string, buffer *utils.Buffer) {
	if db == nil || user == nil || buffer == nil || buffer.IsEmpty() {
		return
	}

	userID := auth.GetId(db, user)
	billing.CreateRecord(
		db, userID, user.Username, "consume",
		buffer.GetTokenName(), model,
		int64(buffer.CountRecordInputToken()), int64(buffer.CountRecordOutputToken()),
		float64(buffer.GetRecordQuota()), buffer.GetDuration(),
		buffer.GetBillingDetail(), buffer.GetRecordPrompts(), buffer.GetRecordResponsePrompts(),
		buffer.GetChannelId(), buffer.GetChannelName(),
	)
}
