package askuser

import (
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	ToolName              = "ask_user"
	AnswerType            = "ask_user_answer"
	MaxQuestions          = 3
	MaxOptionsPerQuestion = 3
	MaxCustomAnswerRunes  = 1000
)

const (
	QuestionTypeSingle   = "single"
	QuestionTypeMultiple = "multiple"
)

type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type Question struct {
	ID       string   `json:"id"`
	Header   string   `json:"header,omitempty"`
	Question string   `json:"question"`
	Type     string   `json:"type"`
	Options  []Option `json:"options"`
}

type ToolInput struct {
	Questions []Question `json:"questions"`
}

type rawToolInput struct {
	Questions []json.RawMessage `json:"questions"`
}

type rawQuestion struct {
	ID       string            `json:"id"`
	Header   string            `json:"header"`
	Question string            `json:"question"`
	Type     string            `json:"type"`
	Options  []json.RawMessage `json:"options"`
}

type rawAnswer struct {
	Type    string          `json:"type"`
	Value   json.RawMessage `json:"value"`
	Custom  bool            `json:"custom"`
	Skipped bool            `json:"skipped"`
}

type rawAnswerPayload struct {
	Type    string               `json:"type"`
	Answers map[string]rawAnswer `json:"answers"`
}

type normalizedAnswer struct {
	Type    string      `json:"type"`
	Value   interface{} `json:"value"`
	Custom  bool        `json:"custom"`
	Skipped bool        `json:"skipped"`
}

type normalizedAnswerPayload struct {
	Type    string                      `json:"type"`
	Answers map[string]normalizedAnswer `json:"answers"`
}

func BuildToolDefinition() *globals.FunctionTools {
	requiredTool := []string{"questions"}
	requiredQuestion := []string{"id", "header", "question", "type", "options"}
	requiredOption := []string{"label", "description"}
	additionalProperties := false

	tools := globals.FunctionTools{
		{
			Type: "function",
			Function: globals.ToolFunction{
				Name:        ToolName,
				Description: "Ask the user for information that is genuinely required before continuing. Call this tool by itself and do not also ask the questions in plain text. Provide 1 to 3 concise questions. For each question, choose single or multiple selection and provide 2 to 3 useful options. The interface automatically provides Other and Skip, so never add those options yourself.",
				Parameters: globals.ToolParameters{
					Type: "object",
					Properties: globals.ToolProperties{
						"questions": {
							"type":        "array",
							"description": "Questions that must be answered before you can continue.",
							"minItems":    1,
							"maxItems":    MaxQuestions,
							"items": map[string]interface{}{
								"type":                 "object",
								"additionalProperties": additionalProperties,
								"properties": map[string]interface{}{
									"id": map[string]interface{}{
										"type":        "string",
										"description": "Short stable identifier unique within this tool call, such as scope or platform.",
									},
									"header": map[string]interface{}{
										"type":        "string",
										"description": "Short label for the question, ideally no more than 12 characters.",
									},
									"question": map[string]interface{}{
										"type":        "string",
										"description": "Complete question shown to the user.",
									},
									"type": map[string]interface{}{
										"type":        "string",
										"enum":        []string{QuestionTypeSingle, QuestionTypeMultiple},
										"description": "Whether the user may select one option or multiple options.",
									},
									"options": map[string]interface{}{
										"type":     "array",
										"minItems": 2,
										"maxItems": MaxOptionsPerQuestion,
										"items": map[string]interface{}{
											"type":                 "object",
											"additionalProperties": additionalProperties,
											"properties": map[string]interface{}{
												"label": map[string]interface{}{
													"type":        "string",
													"description": "Short option label.",
												},
												"description": map[string]interface{}{
													"type":        "string",
													"description": "One sentence explaining the option's impact or tradeoff.",
												},
											},
											"required": requiredOption,
										},
									},
								},
								"required": requiredQuestion,
							},
						},
					},
					Required: &requiredTool,
				},
			},
		},
	}

	return &tools
}

func IsToolCall(call globals.ToolCall) bool {
	return strings.EqualFold(strings.TrimSpace(call.Function.Name), ToolName)
}

func FirstToolCall(calls *globals.ToolCalls) (globals.ToolCall, bool) {
	if calls == nil {
		return globals.ToolCall{}, false
	}
	for _, call := range *calls {
		if IsToolCall(call) {
			return call, true
		}
	}
	return globals.ToolCall{}, false
}

func NormalizeToolCall(call globals.ToolCall) (globals.ToolCall, ToolInput, error) {
	if !IsToolCall(call) {
		return call, ToolInput{}, errors.New("not an ask_user tool call")
	}

	input, err := ParseToolInput(call.Function.Arguments)
	if err != nil {
		return call, ToolInput{}, err
	}

	arguments, err := json.Marshal(input)
	if err != nil {
		return call, ToolInput{}, fmt.Errorf("encode normalized questions: %w", err)
	}
	call.Function.Name = ToolName
	call.Function.Arguments = string(arguments)
	if strings.TrimSpace(call.Id) == "" {
		call.Id = ToolName
	}
	if strings.TrimSpace(call.Type) == "" {
		call.Type = "function"
	}
	return call, input, nil
}

func ParseToolInput(arguments string) (ToolInput, error) {
	var raw rawToolInput
	if err := json.Unmarshal([]byte(arguments), &raw); err != nil {
		return ToolInput{}, fmt.Errorf("invalid ask_user arguments: %w", err)
	}
	if len(raw.Questions) == 0 {
		return ToolInput{}, errors.New("questions must contain at least one question")
	}
	if len(raw.Questions) > MaxQuestions {
		return ToolInput{}, fmt.Errorf("questions must contain at most %d questions", MaxQuestions)
	}

	usedIDs := make(map[string]struct{}, len(raw.Questions))
	questions := make([]Question, 0, len(raw.Questions))
	for index, rawItem := range raw.Questions {
		var item rawQuestion
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return ToolInput{}, fmt.Errorf("question %d is invalid: %w", index+1, err)
		}

		questionText := truncateRunes(strings.TrimSpace(item.Question), 240)
		if questionText == "" {
			return ToolInput{}, fmt.Errorf("question %d text is required", index+1)
		}

		questionType := normalizeQuestionType(item.Type)
		if questionType == "" {
			return ToolInput{}, fmt.Errorf("question %d type must be single or multiple", index+1)
		}

		options, err := normalizeOptions(item.Options)
		if err != nil {
			return ToolInput{}, fmt.Errorf("question %d: %w", index+1, err)
		}

		id := normalizeQuestionID(item.ID)
		if id == "" {
			id = fmt.Sprintf("q%d", index+1)
		}
		if _, exists := usedIDs[id]; exists {
			id = fmt.Sprintf("q%d", index+1)
			for suffix := 2; ; suffix++ {
				if _, exists := usedIDs[id]; !exists {
					break
				}
				id = fmt.Sprintf("q%d_%d", index+1, suffix)
			}
		}
		usedIDs[id] = struct{}{}

		questions = append(questions, Question{
			ID:       id,
			Header:   truncateRunes(strings.TrimSpace(item.Header), 24),
			Question: questionText,
			Type:     questionType,
			Options:  options,
		})
	}

	return ToolInput{Questions: questions}, nil
}

func normalizeQuestionType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case QuestionTypeSingle:
		return QuestionTypeSingle
	case "multi", QuestionTypeMultiple:
		return QuestionTypeMultiple
	default:
		return ""
	}
}

func normalizeQuestionID(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' || char == '-' {
			builder.WriteRune(char)
		}
		if builder.Len() >= 48 {
			break
		}
	}
	return builder.String()
}

func normalizeOptions(rawOptions []json.RawMessage) ([]Option, error) {
	if len(rawOptions) < 2 || len(rawOptions) > MaxOptionsPerQuestion {
		return nil, fmt.Errorf("options must contain between 2 and %d items", MaxOptionsPerQuestion)
	}

	options := make([]Option, 0, len(rawOptions))
	labels := make(map[string]struct{}, len(rawOptions))
	for _, raw := range rawOptions {
		var option Option
		if err := json.Unmarshal(raw, &option); err != nil {
			var label string
			if stringErr := json.Unmarshal(raw, &label); stringErr != nil {
				return nil, errors.New("each option must be an object with label and description")
			}
			option.Label = label
		}

		option.Label = truncateRunes(strings.TrimSpace(option.Label), 80)
		option.Description = truncateRunes(strings.TrimSpace(option.Description), 160)
		if option.Label == "" {
			return nil, errors.New("option label is required")
		}
		if _, exists := labels[option.Label]; exists {
			return nil, fmt.Errorf("duplicate option label %q", option.Label)
		}
		labels[option.Label] = struct{}{}
		options = append(options, option)
	}

	return options, nil
}

func ValidateAnswer(call globals.ToolCall, raw json.RawMessage) (string, error) {
	normalizedCall, input, err := NormalizeToolCall(call)
	if err != nil {
		return "", err
	}
	_ = normalizedCall

	var payload rawAnswerPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("invalid ask_user answer: %w", err)
	}
	if payload.Type != AnswerType {
		return "", fmt.Errorf("answer type must be %q", AnswerType)
	}
	if len(payload.Answers) != len(input.Questions) {
		return "", errors.New("every question must have an answer or be skipped")
	}

	normalized := normalizedAnswerPayload{
		Type:    AnswerType,
		Answers: make(map[string]normalizedAnswer, len(input.Questions)),
	}
	for _, question := range input.Questions {
		answer, exists := payload.Answers[question.ID]
		if !exists {
			return "", fmt.Errorf("missing answer for question %q", question.ID)
		}
		if normalizeQuestionType(answer.Type) != question.Type {
			return "", fmt.Errorf("answer type for question %q does not match", question.ID)
		}
		if answer.Skipped {
			value := interface{}("")
			if question.Type == QuestionTypeMultiple {
				value = []string{}
			}
			normalized.Answers[question.ID] = normalizedAnswer{
				Type:    question.Type,
				Value:   value,
				Custom:  false,
				Skipped: true,
			}
			continue
		}

		normalizedValue, err := validateAnswerValue(question, answer)
		if err != nil {
			return "", fmt.Errorf("question %q: %w", question.ID, err)
		}
		normalized.Answers[question.ID] = normalizedAnswer{
			Type:    question.Type,
			Value:   normalizedValue,
			Custom:  answer.Custom,
			Skipped: false,
		}
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("encode ask_user answer: %w", err)
	}
	return string(encoded), nil
}

func validateAnswerValue(question Question, answer rawAnswer) (interface{}, error) {
	labels := make(map[string]struct{}, len(question.Options))
	for _, option := range question.Options {
		labels[option.Label] = struct{}{}
	}

	if question.Type == QuestionTypeSingle {
		var value string
		if err := json.Unmarshal(answer.Value, &value); err != nil {
			return nil, errors.New("single answer value must be a string")
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("answer cannot be empty")
		}
		if utf8.RuneCountInString(value) > MaxCustomAnswerRunes {
			return nil, fmt.Errorf("answer exceeds %d characters", MaxCustomAnswerRunes)
		}
		if _, exists := labels[value]; !exists && !answer.Custom {
			return nil, errors.New("answer must match an option or be marked custom")
		}
		return value, nil
	}

	var values []string
	if err := json.Unmarshal(answer.Value, &values); err != nil {
		return nil, errors.New("multiple answer value must be an array of strings")
	}
	if len(values) == 0 {
		return nil, errors.New("select at least one answer or skip the question")
	}
	if len(values) > len(question.Options)+1 {
		return nil, errors.New("too many selected answers")
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("answer cannot be empty")
		}
		if utf8.RuneCountInString(value) > MaxCustomAnswerRunes {
			return nil, fmt.Errorf("answer exceeds %d characters", MaxCustomAnswerRunes)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		if _, exists := labels[value]; !exists && !answer.Custom {
			return nil, errors.New("answers must match options or be marked custom")
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil, errors.New("select at least one answer or skip the question")
	}
	return normalized, nil
}

func ErrorMessage(call globals.ToolCall, err error) globals.Message {
	return globals.Message{
		Role: globals.Tool,
		Content: utils.Marshal(map[string]string{
			"status": "error",
			"action": ToolName,
			"error":  err.Error(),
		}),
		ToolCallId: utils.ToPtr(call.Id),
	}
}

func AnswerMessage(call globals.ToolCall, raw json.RawMessage) (globals.Message, error) {
	content, err := ValidateAnswer(call, raw)
	if err != nil {
		return globals.Message{}, err
	}
	return globals.Message{
		Role:       globals.Tool,
		Content:    content,
		ToolCallId: utils.ToPtr(call.Id),
	}, nil
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}
