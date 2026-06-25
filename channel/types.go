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
	CacheHit  *float32           `json:"cache_hit,omitempty" mapstructure:"cache_hit"`
	CacheMiss *float32           `json:"cache_miss,omitempty" mapstructure:"cache_miss"`
	Image     *ImageChargeConfig `json:"image,omitempty" mapstructure:"image"`
	Anonymous bool               `json:"anonymous" mapstructure:"anonymous"`
	Unset     bool               `json:"-" mapstructure:"-"`
}

type ImageChargeConfig struct {
	Mode               string                  `json:"mode,omitempty" mapstructure:"mode"`
	MissingPricePolicy string                  `json:"missing_price_policy,omitempty" mapstructure:"missing_price_policy"`
	Default            float32                 `json:"default,omitempty" mapstructure:"default"`
	Request            float32                 `json:"request,omitempty" mapstructure:"request"`
	Reference          float32                 `json:"reference,omitempty" mapstructure:"reference"`
	Size               map[string]float32      `json:"size,omitempty" mapstructure:"size"`
	Quality            map[string]float32      `json:"quality,omitempty" mapstructure:"quality"`
	Rules              []ImageChargeRule       `json:"rules,omitempty" mapstructure:"rules"`
	Usage              *ImageUsageChargeConfig `json:"usage,omitempty" mapstructure:"usage"`
	OutputCount        int                     `json:"output_count,omitempty" mapstructure:"output_count"`
	BillingUnit        string                  `json:"billing_unit,omitempty" mapstructure:"billing_unit"`
}

type ImageChargeRule struct {
	Size        string  `json:"size,omitempty" mapstructure:"size"`
	Quality     string  `json:"quality,omitempty" mapstructure:"quality"`
	MimeType    string  `json:"mime_type,omitempty" mapstructure:"mime_type"`
	AspectRatio string  `json:"aspect_ratio,omitempty" mapstructure:"aspect_ratio"`
	Quota       float32 `json:"quota,omitempty" mapstructure:"quota"`
}

type ImageUsageChargeConfig struct {
	Input  float32 `json:"input,omitempty" mapstructure:"input"`
	Output float32 `json:"output,omitempty" mapstructure:"output"`
	Image  float32 `json:"image,omitempty" mapstructure:"image"`
}

type ChargeSequence []*Charge

type ChargeManager struct {
	Sequence         ChargeSequence     `json:"sequence"`
	Models           map[string]*Charge `json:"models"`
	NonBillingModels []string           `json:"non_billing_models"`
}
