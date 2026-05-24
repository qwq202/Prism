package utils

import (
	"bytes"
	"chat/globals"
	"fmt"
	"image"
	imagedraw "image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/image/webp"
)

const (
	maxRemoteImageBytes     int64 = 100 * 1024 * 1024
	remoteImageFetchTimeout       = 30 * time.Second
)

var remoteImageHTTPClient = &http.Client{Timeout: remoteImageFetchTimeout}

type Image struct {
	Object  image.Image
	Content string
}
type Images []Image

func remoteImageSizeError(maxBytes int64) error {
	return fmt.Errorf("remote image exceeds %dMB limit", maxBytes/1024/1024)
}

func openRemoteImageResponse(source string) (*http.Response, error) {
	instance, err := neturl.Parse(strings.TrimSpace(source))
	if err != nil || instance == nil || instance.Host == "" {
		return nil, fmt.Errorf("invalid image url")
	}
	if instance.Scheme != "http" && instance.Scheme != "https" {
		return nil, fmt.Errorf("unsupported image url scheme")
	}

	req, err := http.NewRequest(http.MethodGet, instance.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := remoteImageHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		_ = res.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return res, nil
}

func readRemoteImageBytes(source string, maxBytes int64) ([]byte, string, error) {
	res, err := openRemoteImageResponse(source)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.ContentLength > maxBytes {
		return nil, "", remoteImageSizeError(maxBytes)
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return nil, "", remoteImageSizeError(maxBytes)
	}

	contentType := normalizeContentType(res.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = normalizeContentType(http.DetectContentType(data))
	}

	return data, contentType, nil
}

func NewImage(url string) (*Image, error) {
	if strings.HasPrefix(url, "data:image/") {
		data := SafeSplit(url, ",", 2)
		if data[1] == "" {
			return nil, nil
		}

		decoded, err := Base64Decode(data[1])
		if err != nil {
			return nil, err
		}
		if int64(len(decoded)) > maxRemoteImageBytes {
			return nil, remoteImageSizeError(maxRemoteImageBytes)
		}

		img, _, err := image.Decode(bytes.NewReader(decoded))
		if err != nil {
			return nil, err
		}

		return &Image{Object: img, Content: url}, nil
	}

	data, _, err := readRemoteImageBytes(url, maxRemoteImageBytes)
	if err != nil {
		return nil, err
	}

	var img image.Image
	suffix := strings.ToLower(path.Ext(url))
	switch suffix {
	case ".png":
		if img, _, err = image.Decode(bytes.NewReader(data)); err != nil {
			return nil, err
		}
	case ".jpg", ".jpeg":
		if img, err = jpeg.Decode(bytes.NewReader(data)); err != nil {
			return nil, err
		}
	case ".webp":
		if img, err = webp.Decode(bytes.NewReader(data)); err != nil {
			return nil, err
		}
	case ".gif":
		ticks, err := gif.DecodeAll(bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		img = ticks.Image[0]
	}

	return &Image{Object: img, Content: url}, nil
}

func NewImageContent(content string) *Image {
	return &Image{Content: content}
}

func ConvertToBase64(url string) (string, error) {
	if strings.HasPrefix(url, "data:image/") {
		data := strings.Split(url, ",")
		if len(data) != 2 {
			return "", nil
		}
		return data[1], nil
	}

	data, _, err := readRemoteImageBytes(url, maxRemoteImageBytes)
	if err != nil {
		return "", err
	}

	return Base64EncodeBytes(data), nil
}

func IsInternalAttachmentURL(source string) bool {
	if source == "" || strings.HasPrefix(source, "data:image/") {
		return false
	}

	instance, err := neturl.Parse(source)
	if err != nil {
		return false
	}

	return strings.Contains(instance.Path, "/attachments/")
}

func NormalizeImageToVisionDataURL(source string) (string, error) {
	imageObject, err := NewImage(source)
	if err != nil {
		return "", err
	}
	if imageObject == nil || imageObject.Object == nil {
		return "", fmt.Errorf("cannot decode image")
	}

	normalized := imageObject.Object
	bounds := normalized.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return "", fmt.Errorf("invalid image dimensions")
	}

	if width < 8 || height < 8 {
		targetWidth := width
		targetHeight := height
		if targetWidth < 8 {
			targetWidth = 8
		}
		if targetHeight < 8 {
			targetHeight = 8
		}

		canvas := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
		imagedraw.Draw(
			canvas,
			image.Rect(0, 0, width, height),
			normalized,
			bounds.Min,
			imagedraw.Src,
		)
		normalized = canvas
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, normalized); err != nil {
		return "", err
	}

	return fmt.Sprintf("data:image/png;base64,%s", Base64EncodeBytes(buffer.Bytes())), nil
}

func (i *Image) GetWidth() int {
	return i.Object.Bounds().Max.X
}

func (i *Image) GetHeight() int {
	return i.Object.Bounds().Max.Y
}

func (i *Image) GetPixel(x int, y int) (uint32, uint32, uint32, uint32) {
	return i.Object.At(x, y).RGBA()
}

func (i *Image) GetPixelColor(x int, y int) (int, int, int) {
	r, g, b, _ := i.GetPixel(x, y)
	return int(r), int(g), int(b)
}

func (i *Image) CountTokens(model string) int {
	if globals.IsVisionModel(model) {
		// tile size is 512x512
		// the max size of image is 2048x2048
		// the image that is larger than 2048x2048 will be resized in 16 tiles

		x := LimitMax(math.Ceil(float64(i.GetWidth())/512), 4)
		y := LimitMax(math.Ceil(float64(i.GetHeight())/512), 4)
		tiles := int(x) * int(y)

		return 85 + 170*tiles
	}

	return 0
}

func (i *Image) IsBase64() bool {
	return strings.HasPrefix(i.Content, "data:image/")
}

func (i *Image) GetType() string {
	// example: image/jpeg, image/png, image/gif

	if i.IsBase64() {
		t := SafeSplit(i.Content, ";", 2)[0]
		return strings.ReplaceAll(t, "data:", "")
	}

	// example: .jpg, .png, .gif to image/jpeg, image/png, image/gif
	switch strings.ToLower(path.Ext(i.Content)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	default:
		return ""
	}
}

func (i *Image) ToBase64() string {
	if i.IsBase64() {
		return i.Content
	}

	// get url content and convert to base64
	data, err := ConvertToBase64(i.Content)
	if err != nil {
		globals.Warn(fmt.Sprintf("cannot convert image to base64: %s", err.Error()))
		return ""
	}

	return fmt.Sprintf("data:%s;base64,%s", i.GetType(), data)
}

func (i *Image) ToRawBase64() string {
	// example: return /9j/...
	if i.IsBase64() {
		return SafeSplit(i.Content, ",", 2)[1]
	}

	// get url content and convert to base64
	data, err := ConvertToBase64(i.Content)
	if err != nil {
		globals.Warn(fmt.Sprintf("cannot convert image to base64: %s", err.Error()))
		return ""
	}

	return data
}

func DownloadImage(url string, path string) error {
	data, _, err := readRemoteImageBytes(url, maxRemoteImageBytes)
	if err != nil {
		return err
	}

	FileDirSafe(path)
	return os.WriteFile(path, data, 0o644)
}
