package admin

import (
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"fmt"

	"github.com/spf13/viper"
)

type ModelTag []string
type MarketModel struct {
	Id               string   `json:"id" mapstructure:"id" required:"true"`
	Name             string   `json:"name" mapstructure:"name" required:"true"`
	Description      string   `json:"description" mapstructure:"description"`
	Default          bool     `json:"default" mapstructure:"default"`
	HighContext      bool     `json:"high_context" mapstructure:"highcontext"`
	VisionModel      bool     `json:"vision_model" mapstructure:"visionmodel"`
	OCRModel         bool     `json:"ocr_model" mapstructure:"ocrmodel"`
	ReverseModel     bool     `json:"reverse_model" mapstructure:"reversemodel"`
	ReasoningModel   bool     `json:"reasoning_model" mapstructure:"reasoning_model"`
	ReasoningEfforts []string `json:"reasoning_efforts,omitempty" mapstructure:"reasoning_efforts"`
	Avatar           string   `json:"avatar" mapstructure:"avatar"`
	Tag              ModelTag `json:"tag" mapstructure:"tag"`
}

type MarketModelView struct {
	Id                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Default               bool     `json:"default"`
	HighContext           bool     `json:"high_context"`
	VisionModel           bool     `json:"vision_model"`
	DrawingModel          bool     `json:"drawing_model"`
	OCRModel              bool     `json:"ocr_model"`
	ReverseModel          bool     `json:"reverse_model"`
	ReasoningModel        bool     `json:"reasoning_model"`
	ReasoningEfforts      []string `json:"reasoning_efforts,omitempty"`
	ReasoningAvailable    []string `json:"reasoning_available_efforts,omitempty"`
	ReasoningConfigurable bool     `json:"reasoning_configurable"`
	Avatar                string   `json:"avatar"`
	Tag                   ModelTag `json:"tag"`
	ChannelType           string   `json:"channel_type,omitempty"`
}
type MarketModelList []MarketModel

type Market struct {
	Models MarketModelList `json:"models" mapstructure:"models"`
}

const imageGenerationTag = "image-generation"
const reasoningTag = "reasoning"

var defaultCustomReasoningEfforts = []string{"low", "medium", "high"}

var saveMarketConfig = func(models MarketModelList) error {
	return utils.SaveConfig("market", models)
}

func ensureDrawingMarketTag(tags ModelTag, drawingModel bool) ModelTag {
	if !drawingModel {
		return tags
	}
	for _, tag := range tags {
		if tag == imageGenerationTag {
			return tags
		}
	}
	next := append(ModelTag{}, tags...)
	return append(next, imageGenerationTag)
}

func ensureReasoningMarketTag(tags ModelTag, reasoningModel bool) ModelTag {
	if !reasoningModel {
		return tags
	}
	for _, tag := range tags {
		if tag == reasoningTag {
			return tags
		}
	}
	next := append(ModelTag{}, tags...)
	return append(next, reasoningTag)
}

func normalizeMarketModels(models MarketModelList) MarketModelList {
	normalized := make(MarketModelList, len(models))
	for index, model := range models {
		model.ReasoningEfforts = globals.NormalizeCustomReasoningEfforts(model.ReasoningEfforts)
		managedEfforts := globals.ManagedReasoningEfforts(model.Id)
		if globals.HasManagedReasoningCapabilities("", model.Id) {
			model.ReasoningModel = false
			if len(managedEfforts) == 0 {
				model.ReasoningEfforts = nil
			} else if len(model.ReasoningEfforts) == 0 {
				model.ReasoningEfforts = append([]string(nil), managedEfforts...)
			} else {
				model.ReasoningEfforts = intersectMarketReasoningEfforts(
					managedEfforts,
					model.ReasoningEfforts,
				)
				if len(model.ReasoningEfforts) == 0 {
					model.ReasoningEfforts = append([]string(nil), managedEfforts...)
				}
			}
		} else if model.ReasoningModel {
			if len(model.ReasoningEfforts) == 0 {
				model.ReasoningEfforts = append([]string(nil), defaultCustomReasoningEfforts...)
			}
		} else {
			model.ReasoningEfforts = nil
		}
		normalized[index] = model
	}
	return normalized
}

func intersectMarketReasoningEfforts(available []string, selected []string) []string {
	enabled := make(map[string]bool, len(selected))
	for _, effort := range selected {
		enabled[effort] = true
	}

	result := make([]string, 0, len(available))
	for _, effort := range available {
		if enabled[effort] {
			result = append(result, effort)
		}
	}
	return result
}

func syncCustomReasoningCapabilities(models MarketModelList) {
	config := make(map[string][]string)
	for _, model := range models {
		if model.ReasoningModel || len(globals.ManagedReasoningEfforts(model.Id)) > 0 {
			config[model.Id] = model.ReasoningEfforts
		}
	}
	globals.SetCustomReasoningEfforts(config)
}

func NewMarket() *Market {
	var models MarketModelList
	if err := viper.UnmarshalKey("market", &models); err != nil {
		globals.Warn(fmt.Sprintf("[market] read config error: %s, use default config", err.Error()))
		models = MarketModelList{}
	}
	models = normalizeMarketModels(models)
	syncCustomReasoningCapabilities(models)

	return &Market{
		Models: models,
	}
}

func (m *Market) GetModels() MarketModelList {
	return m.Models
}

func (m *Market) GetViewModels() []MarketModelView {
	items := make([]MarketModelView, 0, len(m.Models))

	for _, model := range m.Models {
		channelType := ""
		if channel.ConduitInstance != nil {
			if seq := channel.ConduitInstance.HitSequence(model.Id); len(seq) > 0 && seq[0] != nil {
				channelType = seq[0].GetType()
			}
		}

		drawingModel := globals.IsDrawingModel(channelType, model.Id)
		managedEfforts := globals.ManagedReasoningEfforts(model.Id)
		reasoningConfigurable := !globals.HasManagedReasoningCapabilities(channelType, model.Id)
		reasoningModel := len(managedEfforts) > 0 || (reasoningConfigurable && model.ReasoningModel)
		reasoningEfforts := []string(nil)
		if reasoningModel {
			reasoningEfforts = append(reasoningEfforts, model.ReasoningEfforts...)
		}
		tags := ensureDrawingMarketTag(model.Tag, drawingModel)
		tags = ensureReasoningMarketTag(tags, reasoningModel)

		items = append(items, MarketModelView{
			Id:                    model.Id,
			Name:                  model.Name,
			Description:           model.Description,
			Default:               model.Default,
			HighContext:           model.HighContext,
			VisionModel:           model.VisionModel,
			DrawingModel:          drawingModel,
			OCRModel:              model.OCRModel,
			ReverseModel:          model.ReverseModel,
			ReasoningModel:        reasoningModel,
			ReasoningEfforts:      reasoningEfforts,
			ReasoningAvailable:    append([]string(nil), managedEfforts...),
			ReasoningConfigurable: reasoningConfigurable,
			Avatar:                model.Avatar,
			Tag:                   tags,
			ChannelType:           channelType,
		})
	}

	return items
}

func (m *Market) GetModel(id string) *MarketModel {
	for _, model := range m.Models {
		if model.Id == id {
			return &model
		}
	}
	return nil
}

func (m *Market) SaveConfig() error {
	return saveMarketConfig(m.Models)
}

func (m *Market) SetModels(models MarketModelList) error {
	models = normalizeMarketModels(models)
	if err := saveMarketConfig(models); err != nil {
		return err
	}
	m.Models = models
	syncCustomReasoningCapabilities(models)
	return nil
}
