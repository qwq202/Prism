package personalization

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxCustomInstructionLength = 10000
	maxNicknameLength          = 200
	maxOccupationLength        = 500
	maxAboutUserLength         = 10000
)

type Settings struct {
	PersonaStyle             string `json:"persona_style"`
	PersonaWarmth            string `json:"persona_warmth"`
	PersonaEnthusiasm        string `json:"persona_enthusiasm"`
	PersonaLists             string `json:"persona_lists"`
	PersonaEmoji             string `json:"persona_emoji"`
	PersonaCustomInstruction string `json:"persona_custom_instruction"`
	PersonaNickname          string `json:"persona_nickname"`
	PersonaOccupation        string `json:"persona_occupation"`
	PersonaAboutUser         string `json:"persona_about_user"`
	MemoryEnabled            bool   `json:"memory_enabled"`
	MemoryHistoryEnabled     bool   `json:"memory_history_enabled"`
}

type Record struct {
	Settings  Settings `json:"settings"`
	Revision  int64    `json:"revision"`
	UpdatedAt string   `json:"updated_at"`
}

func normalizeChoice(value, fallback, field string, allowed map[string]struct{}) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if _, ok := allowed[value]; !ok {
		return "", fmt.Errorf("invalid %s", field)
	}
	return value, nil
}

func validateTextLength(value, field string, maxLength int) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("invalid %s", field)
	}
	if utf8.RuneCountInString(value) > maxLength {
		return fmt.Errorf("%s is too long", field)
	}
	return nil
}

func normalizeSettings(value Settings) (Settings, error) {
	var err error
	value.PersonaStyle, err = normalizeChoice(value.PersonaStyle, "default", "persona_style", map[string]struct{}{
		"default": {}, "professional": {}, "friendly": {}, "direct": {}, "creative": {}, "efficient": {}, "sarcastic": {},
	})
	if err != nil {
		return Settings{}, err
	}
	value.PersonaWarmth, err = normalizeChoice(value.PersonaWarmth, "default", "persona_warmth", map[string]struct{}{
		"default": {}, "low": {}, "high": {},
	})
	if err != nil {
		return Settings{}, err
	}
	value.PersonaEnthusiasm, err = normalizeChoice(value.PersonaEnthusiasm, "default", "persona_enthusiasm", map[string]struct{}{
		"default": {}, "low": {}, "high": {},
	})
	if err != nil {
		return Settings{}, err
	}
	value.PersonaLists, err = normalizeChoice(value.PersonaLists, "default", "persona_lists", map[string]struct{}{
		"default": {}, "minimal": {}, "balanced": {}, "structured": {},
	})
	if err != nil {
		return Settings{}, err
	}
	value.PersonaEmoji, err = normalizeChoice(value.PersonaEmoji, "default", "persona_emoji", map[string]struct{}{
		"default": {}, "none": {}, "light": {}, "expressive": {},
	})
	if err != nil {
		return Settings{}, err
	}

	value.PersonaCustomInstruction = strings.TrimSpace(value.PersonaCustomInstruction)
	value.PersonaNickname = strings.TrimSpace(value.PersonaNickname)
	value.PersonaOccupation = strings.TrimSpace(value.PersonaOccupation)
	value.PersonaAboutUser = strings.TrimSpace(value.PersonaAboutUser)

	for _, item := range []struct {
		value string
		field string
		max   int
	}{
		{value.PersonaCustomInstruction, "persona_custom_instruction", maxCustomInstructionLength},
		{value.PersonaNickname, "persona_nickname", maxNicknameLength},
		{value.PersonaOccupation, "persona_occupation", maxOccupationLength},
		{value.PersonaAboutUser, "persona_about_user", maxAboutUserLength},
	} {
		if err := validateTextLength(item.value, item.field, item.max); err != nil {
			return Settings{}, err
		}
	}

	return value, nil
}
