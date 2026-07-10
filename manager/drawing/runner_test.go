package drawing

import (
	"chat/globals"
	"chat/utils"
	"errors"
	"strings"
	"testing"
)

func TestNewDrawingChatPropsDisablesCache(t *testing.T) {
	messages := []globals.Message{{Role: globals.User, Content: "draw a pig"}}
	buffer := &utils.Buffer{}
	responseFormat := map[string]interface{}{"type": "image", "aspect_ratio": "1:1"}
	thinking := map[string]interface{}{"thinking_level": "minimal"}

	props := newDrawingChatProps(
		"gemini-3-pro-image",
		messages,
		responseFormat,
		thinking,
		buffer,
	)

	if !props.DisableCache {
		t.Fatalf("expected drawing requests to disable response caching")
	}
	if props.Model != "gemini-3-pro-image" || props.OriginalModel != "gemini-3-pro-image" {
		t.Fatalf("unexpected drawing model props: %#v", props)
	}
	if props.Buffer != buffer {
		t.Fatalf("expected drawing props to retain the request buffer")
	}
	if props.ResponseFormat == nil || props.Thinking == nil {
		t.Fatalf("expected drawing request options to be preserved")
	}
}

func TestDrawingTaskChunkHookCapturesLatePayloadAfterCancellationRequest(t *testing.T) {
	db := openDrawingWorkspaceTestDB(t)
	task, err := CreateTask(db, 7, validDrawingTaskForm("workspace-hook"))
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := MarkTaskRunning(db, task.TaskID); err != nil {
		t.Fatalf("mark task running: %v", err)
	}

	buffer := &utils.Buffer{}
	hook := newDrawingTaskChunkHook(db, task.TaskID, buffer)
	requested, err := RequestTaskCancellation(db, 7, task.TaskID)
	if err != nil {
		t.Fatalf("request cancellation: %v", err)
	}
	if requested.Status != TaskStatusCanceling {
		t.Fatalf("expected running task to enter canceling, got %q", requested.Status)
	}

	err = hook(&globals.Chunk{Content: "late generated image"})
	if !errors.Is(err, errDrawingTaskCancellationRequested) {
		t.Fatalf("expected cancellation sentinel, got %v", err)
	}
	if !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("cancellation sentinel must stop adapter retry/failover, got %q", err.Error())
	}
	if buffer.Read() != "late generated image" || !buffer.HasVisiblePayload() {
		t.Fatalf("late visible payload must be retained for billing, got %q", buffer.Read())
	}
}

func TestSettleDrawingTaskErrorUsageChargesVisibleAndRevertsEmpty(t *testing.T) {
	visible := &utils.Buffer{}
	visible.Write("generated image payload")

	chargeCalls := 0
	revertCalls := 0
	settleDrawingTaskErrorUsage(
		visible,
		func() { chargeCalls++ },
		func() { revertCalls++ },
	)
	if chargeCalls != 1 || revertCalls != 0 {
		t.Fatalf("visible canceled payload must charge exactly once, charge=%d revert=%d", chargeCalls, revertCalls)
	}

	chargeCalls = 0
	revertCalls = 0
	settleDrawingTaskErrorUsage(
		&utils.Buffer{},
		func() { chargeCalls++ },
		func() { revertCalls++ },
	)
	if chargeCalls != 0 || revertCalls != 1 {
		t.Fatalf("empty canceled request must only revert reservation, charge=%d revert=%d", chargeCalls, revertCalls)
	}
}

func TestRunnerFinalizesCancellationWhenWorkerExits(t *testing.T) {
	db := openDrawingWorkspaceTestDB(t)
	task, err := CreateTask(db, 7, validDrawingTaskForm("workspace-finalize"))
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := MarkTaskRunning(db, task.TaskID); err != nil {
		t.Fatalf("mark task running: %v", err)
	}
	if _, err := RequestTaskCancellation(db, 7, task.TaskID); err != nil {
		t.Fatalf("request cancellation: %v", err)
	}

	finalizeTaskCancellation(db, task.TaskID)

	loaded, err := LoadTask(db, 7, task.TaskID)
	if err != nil {
		t.Fatalf("load finalized task: %v", err)
	}
	if loaded.Status != TaskStatusCanceled || loaded.CompletedAt == "" {
		t.Fatalf("expected worker exit to finalize canceled task, got %#v", loaded)
	}
}
