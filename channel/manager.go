package channel

import (
	"chat/globals"
	"chat/utils"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

var ConduitInstance *Manager
var ChargeInstance *ChargeManager
var SystemInstance *SystemConfig
var PlanInstance *PlanManager

var saveChannelConfig = func(seq Sequence) error {
	return utils.SaveConfig("channel", seq)
}

func InitManager() {
	ConduitInstance = NewChannelManager()
	ChargeInstance = NewChargeManager()
	SystemInstance = NewSystemConfig()
	PlanInstance = NewPlanManager()
}

func NewChannelManager() *Manager {
	var seq Sequence
	if err := viper.UnmarshalKey("channel", &seq); err != nil {
		panic(err)
	}

	if filtered, removed := filterRemovedChannels(seq); removed > 0 {
		globals.Warn(fmt.Sprintf("[channel] removed %d deprecated channel(s) from config", removed))
		seq = filtered
		if err := utils.SaveConfig("channel", seq); err != nil {
			globals.Warn(fmt.Sprintf("[channel] failed to persist deprecated channel cleanup: %s", err.Error()))
		}
	}

	// sort by priority

	manager := &Manager{
		Sequence:          seq,
		Models:            []string{},
		PreflightSequence: map[string]Sequence{},
	}
	manager.Load()

	return manager
}

func (m *Manager) Load() {
	// load channels
	for _, channel := range m.Sequence {
		if channel != nil {
			channel.Load()
		}
	}

	// init support models
	m.Models = []string{}
	for _, channel := range m.GetActiveSequence() {
		for _, model := range channel.GetHitModels() {
			if !utils.Contains(model, m.Models) {
				m.Models = append(m.Models, model)
			}
		}
	}

	// init preflight sequence
	m.PreflightSequence = map[string]Sequence{}
	for _, model := range m.Models {
		var seq Sequence
		for _, channel := range m.GetActiveSequence() {
			if channel.IsHit(model) {
				seq = append(seq, channel)
			}
		}
		seq.Sort()
		m.PreflightSequence[model] = seq
	}

	stamp := time.Now().Unix()

	globals.SupportModels = m.Models
	globals.V1ListModels = globals.ListModels{
		Object: "list",
		Data: utils.Each(m.Models, func(model string) globals.ListModelsItem {
			return globals.ListModelsItem{
				Id:      model,
				Object:  "model",
				Created: stamp,
				OwnedBy: "system",
			}
		}),
	}
}

func (m *Manager) GetSequence() Sequence {
	return m.Sequence
}

func (m *Manager) GetActiveSequence() Sequence {
	var seq Sequence
	for _, channel := range m.Sequence {
		if channel.GetState() {
			seq = append(seq, channel)
		}
	}
	seq.Sort()
	return seq
}

func (m *Manager) GetModels() []string {
	return m.Models
}

func (m *Manager) GetPreflightSequence() map[string]Sequence {
	return m.PreflightSequence
}

// HitSequence returns the preflight sequence of the model
func (m *Manager) HitSequence(model string) Sequence {
	return m.PreflightSequence[model]
}

// HasChannel returns whether the channel exists
func (m *Manager) HasChannel(model string) bool {
	return utils.Contains(model, m.Models)
}

func (m *Manager) GetTicker(model, group string) *Ticker {
	if !m.HasChannel(model) {
		return nil
	}

	return NewTicker(m.HitSequence(model), group)
}

func (m *Manager) Len() int {
	return len(m.Sequence)
}

func (m *Manager) GetMaxId() int {
	var max int
	for _, channel := range m.Sequence {
		if channel.Id > max {
			max = channel.Id
		}
	}
	return max
}

func (m *Manager) SaveConfig() error {
	return saveChannelConfig(m.Sequence)
}

func (m *Manager) CreateChannel(channel *Channel) error {
	if err := validateChannelType(channel); err != nil {
		return err
	}
	channel.Id = m.GetMaxId() + 1
	next := append(append(Sequence(nil), m.Sequence...), channel)
	if err := saveChannelConfig(next); err != nil {
		return err
	}
	m.Sequence = next
	return nil
}

func (m *Manager) UpdateChannel(id int, channel *Channel) error {
	if err := validateChannelType(channel); err != nil {
		return err
	}
	for i, item := range m.Sequence {
		if item.Id == id {
			next := append(Sequence(nil), m.Sequence...)
			next[i] = channel
			if err := saveChannelConfig(next); err != nil {
				return err
			}
			m.Sequence = next
			return nil
		}
	}
	return errors.New("channel not found")
}

func (m *Manager) DeleteChannel(id int) error {
	for i, item := range m.Sequence {
		if item.Id == id {
			next := append(Sequence(nil), m.Sequence[:i]...)
			next = append(next, m.Sequence[i+1:]...)
			if err := saveChannelConfig(next); err != nil {
				return err
			}
			m.Sequence = next
			return nil
		}
	}
	return errors.New("channel not found")
}

func (m *Manager) ActivateChannel(id int) error {
	for i, item := range m.Sequence {
		if item.Id == id {
			next := append(Sequence(nil), m.Sequence...)
			next[i] = cloneChannel(item)
			next[i].State = true
			if err := saveChannelConfig(next); err != nil {
				return err
			}
			m.Sequence = next
			return nil
		}
	}
	return errors.New("channel not found")
}

func (m *Manager) DeactivateChannel(id int) error {
	for i, item := range m.Sequence {
		if item.Id == id {
			next := append(Sequence(nil), m.Sequence...)
			next[i] = cloneChannel(item)
			next[i].State = false
			if err := saveChannelConfig(next); err != nil {
				return err
			}
			m.Sequence = next
			return nil
		}
	}
	return errors.New("channel not found")
}

func cloneChannel(channel *Channel) *Channel {
	if channel == nil {
		return nil
	}
	clone := *channel
	return &clone
}

func filterRemovedChannels(seq Sequence) (Sequence, int) {
	filtered := make(Sequence, 0, len(seq))
	removed := 0
	for _, channel := range seq {
		if channel == nil {
			continue
		}
		if isRemovedChannelType(channel.Type) {
			removed++
			continue
		}
		filtered = append(filtered, channel)
	}

	return filtered, removed
}

func validateChannelType(channel *Channel) error {
	if channel == nil {
		return errors.New("channel is required")
	}

	channelType := channel.Type
	if isRemovedChannelType(channelType) {
		return fmt.Errorf("channel type %s has been removed", channelType)
	}
	if !isKnownChannelType(channelType) {
		return fmt.Errorf("unknown channel type %s", channelType)
	}

	return nil
}

func isKnownChannelType(channelType string) bool {
	switch channelType {
	case globals.OpenAIChannelType,
		globals.OpenAIResponsesChannelType,
		globals.XAIChannelType,
		globals.AzureOpenAIChannelType,
		globals.ClaudeChannelType,
		globals.GLMCodingPlanCNChannelType,
		globals.MiniMaxTokenPlanCNChannelType,
		globals.XiaomiTokenPlanCNChannelType,
		globals.PalmChannelType,
		globals.GeminiEnterpriseAgentPlatformChannelType,
		globals.DeepseekChannelType:
		return true
	default:
		return false
	}
}

func isRemovedChannelType(channelType string) bool {
	switch channelType {
	case "bing",
		"slack",
		"sparkdesk",
		"chatglm",
		"qwen",
		"hunyuan",
		"baichuan",
		"skylark",
		"zhinao",
		"moonshot",
		"groq",
		"coze",
		"dify",
		"midjourney":
		return true
	default:
		return false
	}
}
