package channel

import (
	"chat/globals"
	"chat/utils"
	"strings"

	"github.com/spf13/viper"
)

var saveChargeConfig = func(seq ChargeSequence) error {
	return utils.SaveConfig("charge", seq)
}

func NewChargeManager() *ChargeManager {
	var seq ChargeSequence
	if err := viper.UnmarshalKey("charge", &seq); err != nil {
		panic(err)
	}

	m := &ChargeManager{
		Sequence:         seq,
		Models:           map[string]*Charge{},
		NonBillingModels: []string{},
	}
	m.Load()

	return m
}

func (m *ChargeManager) Load() {
	seq := make(ChargeSequence, 0)
	for _, charge := range m.Sequence {
		if charge == nil {
			continue
		}
		if charge.Id == -1 {
			charge.Id = m.GetMaxId() + 1
		}
		seq = append(seq, charge)
	}
	m.Sequence = seq

	// init support models
	m.Models = map[string]*Charge{}
	for _, charge := range m.Sequence {
		for _, model := range charge.Models {
			if _, ok := m.Models[model]; !ok {
				m.Models[model] = charge
			}
		}
	}

	m.NonBillingModels = []string{}
	for _, charge := range m.Sequence {
		if !charge.IsBilling() {
			for _, model := range charge.Models {
				m.NonBillingModels = append(m.NonBillingModels, model)
			}
		}
	}
}

func (m *ChargeManager) GetModels() map[string]*Charge {
	return m.Models
}

func (m *ChargeManager) GetNonBillingModels() []string {
	return m.NonBillingModels
}

func (m *ChargeManager) IsBilling(model string) bool {
	return !utils.Contains(model, m.NonBillingModels)
}

func (m *ChargeManager) GetCharge(model string) *Charge {
	if charge, ok := m.Models[model]; ok {
		return charge
	}
	return &Charge{
		Type:      globals.NonBilling,
		Anonymous: false,
		Unset:     true,
	}
}

func (m *ChargeManager) SaveConfig() error {
	return saveChargeConfig(m.Sequence)
}

func (m *ChargeManager) GetMaxId() int {
	max := 0
	for _, charge := range m.Sequence {
		if charge.Id > max {
			max = charge.Id
		}
	}
	return max
}

func (m *ChargeManager) AddRawRule(charge *Charge) {
	charge.Id = m.GetMaxId() + 1
	m.Sequence = append(m.Sequence, charge)
}

func (m *ChargeManager) AddRule(charge Charge) error {
	next := m.cloneForMutation()
	next.AddRawRule(&charge)
	if err := saveChargeConfig(next.Sequence); err != nil {
		return err
	}

	m.Sequence = next.Sequence
	m.Load()
	return nil
}

func (m *ChargeManager) UpdateRawRule(charge *Charge) {
	for _, item := range m.Sequence {
		if item.Id == charge.Id {
			*item = *charge
			break
		}
	}
}

func (m *ChargeManager) UpdateRule(charge Charge) error {
	next := m.cloneForMutation()
	next.UpdateRawRule(&charge)
	if err := saveChargeConfig(next.Sequence); err != nil {
		return err
	}

	m.Sequence = next.Sequence
	m.Load()
	return nil
}

func (m *ChargeManager) SetRawRule(charge *Charge) {
	if charge.Id == -1 {
		m.AddRawRule(charge)
	} else {
		m.UpdateRawRule(charge)
	}
}

func (m *ChargeManager) SetRule(charge Charge) error {
	next := m.cloneForMutation()
	next.SetRawRule(&charge)
	if err := saveChargeConfig(next.Sequence); err != nil {
		return err
	}

	m.Sequence = next.Sequence
	m.Load()
	return nil
}

func (m *ChargeManager) DeleteRawRule(id int) {
	for i, item := range m.Sequence {
		if item.Id == id {
			m.Sequence = append(m.Sequence[:i], m.Sequence[i+1:]...)
			break
		}
	}
}

func (m *ChargeManager) DeleteRule(id int) error {
	next := m.cloneForMutation()
	next.DeleteRawRule(id)
	if err := saveChargeConfig(next.Sequence); err != nil {
		return err
	}

	m.Sequence = next.Sequence
	m.Load()
	return nil
}

func (m *ChargeManager) SyncRules(charge ChargeSequence, overwrite bool) error {
	next := m.cloneForMutation()
	for _, item := range charge {
		next.SyncRule(cloneChargeRule(item), overwrite)
	}

	if err := saveChargeConfig(next.Sequence); err != nil {
		return err
	}

	m.Sequence = next.Sequence
	m.Load()
	return nil
}

func (m *ChargeManager) SyncRule(charge *Charge, overwrite bool) {
	if overwrite {
		m.SyncRuleWithOverwrite(charge)
	} else {
		m.SyncRuleWithoutOverwrite(charge)
	}
}

func (m *ChargeManager) SyncRuleWithOverwrite(charge *Charge) {
	if len(charge.Models) == 0 {
		return
	}

	for _, model := range charge.GetModels() {
		if raw := m.GetRuleByModel(model); raw != nil {
			if len(raw.Models) == 1 {
				// rule is already exist and only contains this model, just delete it

				m.DeleteRawRule(raw.Id)
			} else {
				// rule is already exist and contains other models, delete this model from it and add a new rule
				// delete model from raw rule
				raw.Models = utils.Filter(raw.Models, func(m string) bool {
					return m != model
				})
				m.UpdateRawRule(raw)
			}
		}
	}

	instance := charge.New("")
	instance.Models = charge.Models
	m.AddRawRule(instance)
}

func (m *ChargeManager) SyncRuleWithoutOverwrite(charge *Charge) {
	models := utils.Filter(charge.GetModels(), func(model string) bool {
		return !m.Contains(model)
	})

	if len(models) > 0 {
		charge.Models = models
		m.AddRawRule(charge)
	}
}

func (m *ChargeManager) ListRules() ChargeSequence {
	return m.Sequence
}

func (m *ChargeManager) cloneForMutation() *ChargeManager {
	return &ChargeManager{
		Sequence: cloneChargeSequence(m.Sequence),
	}
}

func cloneChargeSequence(seq ChargeSequence) ChargeSequence {
	clone := make(ChargeSequence, 0, len(seq))
	for _, item := range seq {
		clone = append(clone, cloneChargeRule(item))
	}
	return clone
}

func cloneChargeRule(item *Charge) *Charge {
	if item == nil {
		return nil
	}
	copied := *item
	copied.Models = append([]string(nil), item.Models...)
	copied.CacheHit = cloneFloat32Ptr(item.CacheHit)
	copied.CacheMiss = cloneFloat32Ptr(item.CacheMiss)
	copied.Image = cloneImageChargeConfig(item.Image)
	return &copied
}

func cloneFloat32Ptr(value *float32) *float32 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func cloneImageChargeConfig(item *ImageChargeConfig) *ImageChargeConfig {
	if item == nil {
		return nil
	}

	copied := *item
	if item.Size != nil {
		copied.Size = make(map[string]float32, len(item.Size))
		for key, value := range item.Size {
			copied.Size[key] = value
		}
	}
	if item.Quality != nil {
		copied.Quality = make(map[string]float32, len(item.Quality))
		for key, value := range item.Quality {
			copied.Quality[key] = value
		}
	}
	return &copied
}

func (m *ChargeManager) Contains(model string) bool {
	for _, item := range m.Sequence {
		if item.Contains(model) {
			return true
		}
	}
	return false
}

func (m *ChargeManager) GetRule(id int) *Charge {
	for _, item := range m.Sequence {
		if item.Id == id {
			return item
		}
	}
	return nil
}

func (m *ChargeManager) GetRuleByModel(model string) *Charge {
	for _, item := range m.Sequence {
		if item.Contains(model) {
			return item
		}
	}
	return nil
}

func (c *Charge) IsUnsetType() bool {
	return c.Unset
}

func (c *Charge) GetType() string {
	if c.Type == "" {
		return globals.NonBilling
	}
	return c.Type
}

func (c *Charge) GetModels() []string {
	return c.Models
}

func (c *Charge) GetInput() float32 {
	if c.Input <= 0 {
		return 0
	}
	return c.Input
}

func (c *Charge) GetOutput() float32 {
	if c.Output <= 0 {
		return 0
	}
	return c.Output
}

func (c *Charge) GetCacheHit() (float32, bool) {
	if c.CacheHit == nil || *c.CacheHit < 0 {
		return 0, false
	}
	return *c.CacheHit, true
}

func (c *Charge) GetCacheMiss() (float32, bool) {
	if c.CacheMiss == nil || *c.CacheMiss < 0 {
		return 0, false
	}
	return *c.CacheMiss, true
}

func (c *Charge) SupportAnonymous() bool {
	return c.Anonymous
}

func (c *Charge) IsBilling() bool {
	return c.GetType() != globals.NonBilling
}

func (c *Charge) IsBillingType(t string) bool {
	return c.GetType() == t
}

func (c *Charge) GetLimit() float32 {
	switch c.GetType() {
	case globals.NonBilling:
		return 0
	case globals.TimesBilling:
		return c.GetOutput()
	case globals.TokenBilling:
		// 1k input tokens + 1k output tokens
		return c.GetInput() + c.GetOutput()
	case globals.ImageBilling:
		return c.EstimateImageQuota(0, nil, 1)
	default:
		return 0
	}
}

func (c *Charge) GetImageChargeConfig() ImageChargeConfig {
	if c.Image == nil {
		return ImageChargeConfig{}
	}
	return *c.Image
}

func normalizeImageBillingKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func getResponseFormatString(data interface{}, keys ...string) string {
	values, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}

	for _, key := range keys {
		if value, ok := values[key]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}

	return ""
}

func lookupImagePrice(prices map[string]float32, value string) (float32, bool) {
	normalized := normalizeImageBillingKey(value)
	if normalized == "" || len(prices) == 0 {
		return 0, false
	}

	for key, price := range prices {
		if normalizeImageBillingKey(key) == normalized {
			return price, true
		}
	}
	return 0, false
}

func (c *Charge) imageUnitPrice(responseFormat interface{}) float32 {
	config := c.GetImageChargeConfig()
	price := config.Default
	if price <= 0 {
		price = c.GetOutput()
	}

	size := getResponseFormatString(responseFormat, "image_size", "imageSize", "size")
	if sizePrice, ok := lookupImagePrice(config.Size, size); ok {
		price = sizePrice
	}

	quality := getResponseFormatString(responseFormat, "quality", "image_quality", "imageQuality")
	if qualityPrice, ok := lookupImagePrice(config.Quality, quality); ok {
		price += qualityPrice
	}

	if price < 0 {
		return 0
	}
	return price
}

func (c *Charge) EstimateImageQuota(referenceImages int, responseFormat interface{}, outputImages int) float32 {
	if c == nil || c.GetType() != globals.ImageBilling {
		return 0
	}
	if outputImages <= 0 {
		return 0
	}
	if referenceImages < 0 {
		referenceImages = 0
	}

	config := c.GetImageChargeConfig()
	quota := config.Request
	quota += c.imageUnitPrice(responseFormat) * float32(outputImages)
	quota += config.Reference * float32(referenceImages)
	if quota < 0 {
		return 0
	}
	return quota
}

func (c *Charge) GetImageBillingDetail(referenceImages int, responseFormat interface{}, outputImages int) map[string]interface{} {
	config := c.GetImageChargeConfig()
	detail := map[string]interface{}{
		"billing_unit":     utils.Multi(config.BillingUnit != "", config.BillingUnit, "final_image"),
		"output_images":    outputImages,
		"reference_images": referenceImages,
		"unit_quota":       c.imageUnitPrice(responseFormat),
		"request_quota":    config.Request,
		"reference_quota":  config.Reference,
	}

	if size := getResponseFormatString(responseFormat, "image_size", "imageSize", "size"); size != "" {
		detail["image_size"] = size
	}
	if quality := getResponseFormatString(responseFormat, "quality", "image_quality", "imageQuality"); quality != "" {
		detail["quality"] = quality
	}
	if mimeType := getResponseFormatString(responseFormat, "mime_type", "mimeType"); mimeType != "" {
		detail["mime_type"] = mimeType
	}
	if aspectRatio := getResponseFormatString(responseFormat, "aspect_ratio", "aspectRatio"); aspectRatio != "" {
		detail["aspect_ratio"] = aspectRatio
	}

	return detail
}

func (c *Charge) Contains(model string) bool {
	return utils.Contains(model, c.Models)
}

func (c *Charge) New(model string) *Charge {
	return &Charge{
		Type:      c.Type,
		Models:    []string{model},
		Input:     c.Input,
		Output:    c.Output,
		CacheHit:  cloneFloat32Ptr(c.CacheHit),
		CacheMiss: cloneFloat32Ptr(c.CacheMiss),
		Image:     cloneImageChargeConfig(c.Image),
		Anonymous: c.Anonymous,
	}
}
