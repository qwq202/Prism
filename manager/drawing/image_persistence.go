package drawing

import (
	"chat/utils"
	"fmt"
	neturl "net/url"
	"strings"
)

var persistRemoteDrawingImageSource = utils.PersistImageSource

func persistGeneratedImageSource(source string) (string, error) {
	stored, persisted, err := storeDrawingDataURL(strings.TrimSpace(source))
	if err != nil {
		return "", err
	}
	if persisted || utils.IsInternalAttachmentURL(stored) {
		return stored, nil
	}

	instance, err := neturl.Parse(stored)
	if err != nil {
		return "", fmt.Errorf("parse generated image source: %w", err)
	}
	scheme := strings.ToLower(instance.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("unsupported generated image source scheme: %s", scheme)
	}

	stored, err = persistRemoteDrawingImageSource(stored)
	if err != nil {
		return "", fmt.Errorf("persist generated image source: %w", err)
	}
	return stored, nil
}
