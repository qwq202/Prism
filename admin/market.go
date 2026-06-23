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
	Id           string   `json:"id" mapstructure:"id" required:"true"`
	Name         string   `json:"name" mapstructure:"name" required:"true"`
	Description  string   `json:"description" mapstructure:"description"`
	Default      bool     `json:"default" mapstructure:"default"`
	HighContext  bool     `json:"high_context" mapstructure:"highcontext"`
	VisionModel  bool     `json:"vision_model" mapstructure:"visionmodel"`
	OCRModel     bool     `json:"ocr_model" mapstructure:"ocrmodel"`
	ReverseModel bool     `json:"reverse_model" mapstructure:"reversemodel"`
	Avatar       string   `json:"avatar" mapstructure:"avatar"`
	Tag          ModelTag `json:"tag" mapstructure:"tag"`
}

type MarketModelView struct {
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Default      bool     `json:"default"`
	HighContext  bool     `json:"high_context"`
	VisionModel  bool     `json:"vision_model"`
	DrawingModel bool     `json:"drawing_model"`
	OCRModel     bool     `json:"ocr_model"`
	ReverseModel bool     `json:"reverse_model"`
	Avatar       string   `json:"avatar"`
	Tag          ModelTag `json:"tag"`
	ChannelType  string   `json:"channel_type,omitempty"`
}
type MarketModelList []MarketModel

type Market struct {
	Models MarketModelList `json:"models" mapstructure:"models"`
}

var saveMarketConfig = func(models MarketModelList) error {
	return utils.SaveConfig("market", models)
}

func NewMarket() *Market {
	var models MarketModelList
	if err := viper.UnmarshalKey("market", &models); err != nil {
		globals.Warn(fmt.Sprintf("[market] read config error: %s, use default config", err.Error()))
		models = MarketModelList{}
	}

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

		items = append(items, MarketModelView{
			Id:           model.Id,
			Name:         model.Name,
			Description:  model.Description,
			Default:      model.Default,
			HighContext:  model.HighContext,
			VisionModel:  model.VisionModel,
			DrawingModel: globals.IsDrawingModel(channelType, model.Id),
			OCRModel:     model.OCRModel,
			ReverseModel: model.ReverseModel,
			Avatar:       model.Avatar,
			Tag:          model.Tag,
			ChannelType:  channelType,
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
	if err := saveMarketConfig(models); err != nil {
		return err
	}
	m.Models = models
	return nil
}
