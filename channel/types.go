package channel

import (
	"chat/globals"
)

type Channel struct {
	Id            int                 `json:"id" mapstructure:"id"`
	Name          string              `json:"name" mapstructure:"name"`
	Type          string              `json:"type" mapstructure:"type"`
	Priority      int                 `json:"priority" mapstructure:"priority"`
	Weight        int                 `json:"weight" mapstructure:"weight"`
	Models        []string            `json:"models" mapstructure:"models"`
	Retry         int                 `json:"retry" mapstructure:"retry"`
	Secret        string              `json:"secret" mapstructure:"secret"`
	Endpoint      string              `json:"endpoint" mapstructure:"endpoint"`
	Mapper        string              `json:"mapper" mapstructure:"mapper"`
	State         bool                `json:"state" mapstructure:"state"`
	Group         []string            `json:"group" mapstructure:"group"`
	Proxy         globals.ProxyConfig `json:"proxy" mapstructure:"proxy"`
	Reflect       *map[string]string  `json:"-"`
	HitModels     *[]string           `json:"-"`
	ExcludeModels *[]string           `json:"-"`
	CurrentSecret *string             `json:"-"`
}

type Sequence []*Channel

type Manager struct {
	Sequence          Sequence            `json:"sequence"`
	PreflightSequence map[string]Sequence `json:"preflight_sequence"`
	Models            []string            `json:"models"`
}

type Ticker struct {
	Sequence Sequence `json:"sequence"`
	Cursor   int      `json:"cursor"`
}

type Charge struct {
	Id        int                `json:"id" mapstructure:"id"`
	Type      string             `json:"type" mapstructure:"type"`
	Models    []string           `json:"models" mapstructure:"models"`
	Input     float32            `json:"input" mapstructure:"input"`
	Output    float32            `json:"output" mapstructure:"output"`
	Image     *ImageChargeConfig `json:"image,omitempty" mapstructure:"image"`
	Anonymous bool               `json:"anonymous" mapstructure:"anonymous"`
	Unset     bool               `json:"-" mapstructure:"-"`
}

type ImageChargeConfig struct {
	Default     float32            `json:"default,omitempty" mapstructure:"default"`
	Request     float32            `json:"request,omitempty" mapstructure:"request"`
	Reference   float32            `json:"reference,omitempty" mapstructure:"reference"`
	Size        map[string]float32 `json:"size,omitempty" mapstructure:"size"`
	Quality     map[string]float32 `json:"quality,omitempty" mapstructure:"quality"`
	OutputCount int                `json:"output_count,omitempty" mapstructure:"output_count"`
	BillingUnit string             `json:"billing_unit,omitempty" mapstructure:"billing_unit"`
}

type ChargeSequence []*Charge

type ChargeManager struct {
	Sequence         ChargeSequence     `json:"sequence"`
	Models           map[string]*Charge `json:"models"`
	NonBillingModels []string           `json:"non_billing_models"`
}
