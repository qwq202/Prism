package adaptercommon

import "chat/utils"

type ImageInputMode = utils.ImageInputMode
type ImageInputCapability = utils.ImageInputCapability
type NormalizedImageInput = utils.NormalizedImageInput

const (
	ImageInputModeURL           ImageInputMode = utils.ImageInputModeURL
	ImageInputModeVisionDataURL ImageInputMode = utils.ImageInputModeVisionDataURL
	ImageInputModeInlineBase64  ImageInputMode = utils.ImageInputModeInlineBase64
)

var (
	URLImageInputCapability           = utils.URLImageInputCapability
	VisionDataURLImageInputCapability = utils.VisionDataURLImageInputCapability
	InlineBase64ImageInputCapability  = utils.InlineBase64ImageInputCapability
)

func NormalizeImageForCapability(source string, capability ImageInputCapability) (*NormalizedImageInput, error) {
	return utils.NormalizeImageForCapability(source, capability)
}
