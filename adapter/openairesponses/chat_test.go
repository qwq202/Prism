package openairesponses

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"testing"
)

func TestEmitReasoningSummaryStartsThinkBlock(t *testing.T) {
	started := false

	chunk := emitReasoningSummary("step 1", &started)
	if chunk == nil {
		t.Fatalf("expected reasoning chunk")
	}
	if chunk.Content != "<think>\nstep 1" {
		t.Fatalf("unexpected reasoning chunk content: %q", chunk.Content)
	}
	if !started {
		t.Fatalf("expected reasoning block to be marked as started")
	}
}

func TestEmitOutputTextClosesThinkBlockBeforeAnswer(t *testing.T) {
	reasoningStarted := true
	reasoningClosed := false

	chunk := emitOutputText("final answer", &reasoningStarted, &reasoningClosed)
	if chunk == nil {
		t.Fatalf("expected output chunk")
	}
	if chunk.Content != "\n</think>\n\nfinal answer" {
		t.Fatalf("unexpected output chunk content: %q", chunk.Content)
	}
	if !reasoningClosed {
		t.Fatalf("expected reasoning block to be closed")
	}
}

func TestEmitOutputTextWithoutReasoningLeavesAnswerUntouched(t *testing.T) {
	reasoningStarted := false
	reasoningClosed := false

	chunk := emitOutputText("final answer", &reasoningStarted, &reasoningClosed)
	if chunk == nil {
		t.Fatalf("expected output chunk")
	}
	if chunk.Content != "final answer" {
		t.Fatalf("unexpected output chunk content: %q", chunk.Content)
	}
}

func TestGetChatBodyDisablesStoreForXAIImageRequests(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model:       "grok-4-1-fast-reasoning",
		ChannelType: globals.XAIChannelType,
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "![image](https://example.com/test.png)\n这是什么",
			},
		},
	}

	body := instance.GetChatBody(props)
	if body.Store == nil || *body.Store {
		t.Fatalf("expected xai image requests to set store=false")
	}
	if len(body.Input) != 1 || len(body.Input[0].Content) != 2 {
		t.Fatalf("unexpected input payload shape: %#v", body.Input)
	}
	if body.Input[0].Content[1].Detail == nil || *body.Input[0].Content[1].Detail != "high" {
		t.Fatalf("expected input image detail to be high")
	}
}

func TestGetChatBodyLeavesStoreUnsetForTextOnlyXAIRequests(t *testing.T) {
	instance := &ChatInstance{}
	props := &adaptercommon.ChatProps{
		Model:       "grok-4-1-fast-reasoning",
		ChannelType: globals.XAIChannelType,
		Message: []globals.Message{
			{
				Role:    globals.User,
				Content: "你好",
			},
		},
	}

	body := instance.GetChatBody(props)
	if body.Store != nil {
		t.Fatalf("expected text-only xai requests to leave store unset")
	}
}
