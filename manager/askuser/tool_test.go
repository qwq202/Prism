package askuser

import (
	"chat/globals"
	"encoding/json"
	"testing"
)

func validToolCall() globals.ToolCall {
	return globals.ToolCall{
		Id:   "call_ask_1",
		Type: "function",
		Function: globals.ToolCallFunction{
			Name: ToolName,
			Arguments: `{
				"questions": [
					{
						"id": "scope",
						"header": "Scope",
						"question": "Which scope should be implemented?",
						"type": "single",
						"options": [
							{"label": "Minimal", "description": "Only the core flow."},
							{"label": "Complete", "description": "Include recovery and tests."}
						]
					},
					{
						"id": "parts",
						"header": "Parts",
						"question": "Which parts should be included?",
						"type": "multiple",
						"options": ["UI", "Tests"]
					}
				]
			}`,
		},
	}
}

func TestBuildToolDefinitionDeclaresQuestionSelectionType(t *testing.T) {
	tools := BuildToolDefinition()
	if tools == nil || len(*tools) != 1 {
		t.Fatalf("expected one ask_user tool, got %#v", tools)
	}
	tool := (*tools)[0]
	if tool.Function.Name != ToolName {
		t.Fatalf("unexpected tool name %q", tool.Function.Name)
	}

	encoded, err := json.Marshal(tool.Function.Parameters)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	questions := schema["properties"].(map[string]interface{})["questions"].(map[string]interface{})
	item := questions["items"].(map[string]interface{})
	questionType := item["properties"].(map[string]interface{})["type"].(map[string]interface{})
	enum := questionType["enum"].([]interface{})
	if len(enum) != 2 || enum[0] != QuestionTypeSingle || enum[1] != QuestionTypeMultiple {
		t.Fatalf("unexpected question type enum: %#v", enum)
	}
}

func TestNormalizeToolCallSupportsSingleAndMultipleQuestions(t *testing.T) {
	call, input, err := NormalizeToolCall(validToolCall())
	if err != nil {
		t.Fatalf("normalize tool call: %v", err)
	}
	if len(input.Questions) != 2 {
		t.Fatalf("expected two questions, got %#v", input.Questions)
	}
	if input.Questions[0].Type != QuestionTypeSingle || input.Questions[1].Type != QuestionTypeMultiple {
		t.Fatalf("unexpected normalized question types: %#v", input.Questions)
	}
	if input.Questions[1].Options[0].Label != "UI" {
		t.Fatalf("expected string option compatibility, got %#v", input.Questions[1].Options)
	}
	if call.Function.Arguments == validToolCall().Function.Arguments {
		t.Fatalf("expected normalized arguments to be encoded")
	}
}

func TestValidateAnswerAcceptsSelectionsCustomValuesAndSkips(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "ask_user_answer",
		"answers": {
			"scope": {"type":"single","value":"Complete","custom":false,"skipped":false},
			"parts": {"type":"multiple","value":["UI","Documentation"],"custom":true,"skipped":false}
		}
	}`)

	content, err := ValidateAnswer(validToolCall(), raw)
	if err != nil {
		t.Fatalf("validate answer: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		t.Fatalf("decode normalized result: %v", err)
	}
	if result["type"] != AnswerType {
		t.Fatalf("unexpected result type: %#v", result)
	}
}

func TestValidateAnswerRejectsQuestionTypeMismatch(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "ask_user_answer",
		"answers": {
			"scope": {"type":"multiple","value":["Complete"],"custom":false,"skipped":false},
			"parts": {"type":"multiple","value":[],"custom":false,"skipped":true}
		}
	}`)

	if _, err := ValidateAnswer(validToolCall(), raw); err == nil {
		t.Fatalf("expected mismatched answer type to fail")
	}
}

func TestParseToolInputRejectsMoreThanMaximumQuestions(t *testing.T) {
	_, err := ParseToolInput(`{
		"questions": [
			{"id":"q1","question":"Q1?","type":"single","options":["A","B"]},
			{"id":"q2","question":"Q2?","type":"single","options":["A","B"]},
			{"id":"q3","question":"Q3?","type":"single","options":["A","B"]},
			{"id":"q4","question":"Q4?","type":"single","options":["A","B"]},
			{"id":"q5","question":"Q5?","type":"single","options":["A","B"]}
		]
	}`)
	if err == nil {
		t.Fatalf("expected too many questions to fail")
	}
}
