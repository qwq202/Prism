package palm2

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"strings"
	"testing"
)

func TestGetGeminiChatBodyIncludesCodeExecutionTool(t *testing.T) {
	instance := &ChatInstance{}
	body := instance.GetGeminiChatBody(&adaptercommon.ChatProps{
		Model:               globals.Gemini35Flash,
		Message:             []globals.Message{{Role: globals.User, Content: "sum primes"}},
		EnableCodeExecution: true,
	})

	if len(body.Tools) != 1 || body.Tools[0].CodeExecution == nil {
		t.Fatalf("expected code execution tool, got %#v", body.Tools)
	}
}

func TestGetGeminiChatBodyOmitsCodeExecutionToolByDefault(t *testing.T) {
	instance := &ChatInstance{}
	body := instance.GetGeminiChatBody(&adaptercommon.ChatProps{
		Model:   globals.Gemini35Flash,
		Message: []globals.Message{{Role: globals.User, Content: "hello"}},
	})

	if len(body.Tools) != 0 {
		t.Fatalf("expected no built-in tools by default, got %#v", body.Tools)
	}
}

func TestGetGeminiChatBodyOmitsCodeExecutionToolForUnsupportedModel(t *testing.T) {
	instance := &ChatInstance{}
	body := instance.GetGeminiChatBody(&adaptercommon.ChatProps{
		Model:               globals.Gemini3ProImagePreview,
		Message:             []globals.Message{{Role: globals.User, Content: "draw image"}},
		EnableCodeExecution: true,
	})

	if len(body.Tools) != 0 {
		t.Fatalf("expected no code execution tool for unsupported model, got %#v", body.Tools)
	}
}

func TestGetGeminiChatTextIncludesCodeExecutionParts(t *testing.T) {
	instance := &ChatInstance{}
	text := instance.GetGeminiChatText(globals.Gemini35Flash, []GeminiChatPart{
		{Text: ptrString("I calculated it.")},
		{
			ExecutableCode: &GeminiExecutableCode{
				Language: "PYTHON",
				Code:     "print(2 + 2)",
			},
		},
		{
			CodeExecutionResult: &GeminiCodeExecutionResult{
				Outcome: "OUTCOME_OK",
				Output:  "4",
			},
		},
	})

	for _, expected := range []string{"I calculated it.", "```python", "print(2 + 2)", "```text", "4"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in rendered Gemini text, got %q", expected, text)
		}
	}
}

func TestGetGeminiChunkMarksCodeExecutionUsed(t *testing.T) {
	instance := &ChatInstance{}
	chunk, err := instance.GetGeminiChunk(globals.Gemini35Flash, GeminiChatResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Parts: []GeminiChatPart{
						{
							ExecutableCode: &GeminiExecutableCode{
								Language: "PYTHON",
								Code:     "print(2 + 2)",
							},
						},
						{
							CodeExecutionResult: &GeminiCodeExecutionResult{
								Outcome: "OUTCOME_OK",
								Output:  "4",
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chunk.BuiltinToolUsage == nil || chunk.BuiltinToolUsage.CodeExecution == nil {
		t.Fatalf("expected code execution usage metadata, got %#v", chunk.BuiltinToolUsage)
	}
	if !chunk.BuiltinToolUsage.CodeExecution.Used {
		t.Fatalf("expected code execution used flag, got %#v", chunk.BuiltinToolUsage.CodeExecution)
	}
}
