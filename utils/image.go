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

	"golang.org/x/image/webp"
)

type Image struct {
	Object  image.Image
	Content string
}
type Images []Image

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

		img, _, err := image.Decode(strings.NewReader(string(decoded)))
		if err != nil {
			return nil, err
		}

		return &Image{Object: img, Content: url}, nil
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	var img image.Image
	suffix := strings.ToLower(path.Ext(url))
	switch suffix {
	case ".png":
		if img, _, err = image.Decode(res.Body); err != nil {
			return nil, err
		}
	case ".jpg", ".jpeg":
		if img, err = jpeg.Decode(res.Body); err != nil {
			return nil, err
		}
	case ".webp":
		if img, err = webp.Decode(res.Body); err != nil {
			return nil, err
		}
	case ".gif":
		ticks, err := gif.DecodeAll(res.Body)
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

	res, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
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
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			globals.Debug(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(res.Body)

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			globals.Debug(fmt.Sprintf("[utils] close file error: %s (path: %s)", err.Error(), path))
		}
	}(file)

	_, err = io.Copy(file, res.Body)
	return err
}
