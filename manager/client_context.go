package manager

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func extractClientContext(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	ua := strings.TrimSpace(c.Request.UserAgent())
	if ua == "" {
		return ""
	}

	deviceType := detectDeviceType(ua)
	operatingSystem := detectOperatingSystem(ua)
	browser := detectBrowser(ua)

	parts := make([]string, 0, 3)
	if operatingSystem != "" && operatingSystem != "Unknown" {
		parts = append(parts, fmt.Sprintf("Operating system: %s", operatingSystem))
	}
	if browser != "" && browser != "Unknown" {
		parts = append(parts, fmt.Sprintf("Browser/App: %s", browser))
	}
	if deviceType != "" && deviceType != "Unknown" {
		parts = append(parts, fmt.Sprintf("Device type: %s", deviceType))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "; ") + "."
}

func detectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)

	if containsAny(ua, "ipad", "tablet", "playbook", "kindle", "silk/", "sm-t", "nexus 7", "nexus 9", "xoom") {
		return "tablet"
	}
	if containsAny(ua, "iphone", "ipod", "android", "windows phone", "mobile", "mobi") {
		return "phone"
	}
	if containsAny(ua, "macintosh", "windows nt", "x11", "linux", "cros") {
		return "computer"
	}

	return "Unknown"
}

func detectOperatingSystem(userAgent string) string {
	ua := strings.ToLower(userAgent)

	switch {
	case containsAny(ua, "iphone", "ipad", "cpu iphone os", "cpu os"):
		return "iOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "cros"):
		return "ChromeOS"
	case strings.Contains(ua, "windows nt 10.0"):
		return "Windows 10/11"
	case strings.Contains(ua, "windows nt 6.3"):
		return "Windows 8.1"
	case strings.Contains(ua, "windows nt 6.2"):
		return "Windows 8"
	case strings.Contains(ua, "windows nt 6.1"):
		return "Windows 7"
	case strings.Contains(ua, "windows nt"):
		return "Windows"
	case containsAny(ua, "mac os x", "macintosh"):
		return "macOS"
	case containsAny(ua, "linux", "x11"):
		return "Linux"
	default:
		return "Unknown"
	}
}

func detectBrowser(userAgent string) string {
	ua := strings.ToLower(userAgent)

	switch {
	case strings.Contains(ua, "micromessenger"):
		return "WeChat"
	case strings.Contains(ua, "qqbrowser"):
		return "QQ Browser"
	case containsAny(ua, "alipayclient", "aliapp"):
		return "Alipay"
	case strings.Contains(ua, "electron/"):
		return "Electron"
	case strings.Contains(ua, "edg/"):
		return "Microsoft Edge"
	case containsAny(ua, "opr/", "opera"):
		return "Opera"
	case containsAny(ua, "firefox/", "fxios"):
		return "Firefox"
	case containsAny(ua, "crios", "chrome/"):
		return "Chrome"
	case strings.Contains(ua, "safari/") && strings.Contains(ua, "version/"):
		return "Safari"
	case strings.Contains(ua, "postmanruntime/"):
		return "Postman"
	case strings.Contains(ua, "curl/"):
		return "curl"
	default:
		return "Unknown"
	}
}

func containsAny(value string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}

	return false
}
