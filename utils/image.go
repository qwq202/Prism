package utils

import (
	"bytes"
	"chat/globals"
	"context"
	"fmt"
	"image"
	imagedraw "image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
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

var remoteImageHTTPClient = newRemoteImageHTTPClient()

type Image struct {
	Object  image.Image
	Content string
}
type Images []Image

type ImageInputMode string

const (
	ImageInputModeURL           ImageInputMode = "url"
	ImageInputModeVisionDataURL ImageInputMode = "vision_data_url"
	ImageInputModeInlineBase64  ImageInputMode = "inline_base64"
)

type ImageInputCapability struct {
	Mode ImageInputMode
}

var (
	URLImageInputCapability           = ImageInputCapability{Mode: ImageInputModeURL}
	VisionDataURLImageInputCapability = ImageInputCapability{Mode: ImageInputModeVisionDataURL}
	InlineBase64ImageInputCapability  = ImageInputCapability{Mode: ImageInputModeInlineBase64}
)

type NormalizedImageInput struct {
	Source    string
	MIMEType  string
	RawBase64 string
	Mode      ImageInputMode
}

func remoteImageSizeError(maxBytes int64) error {
	return fmt.Errorf("remote image exceeds %dMB limit", maxBytes/1024/1024)
}

func isUnsafeRemoteImageAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() {
		return true
	}

	if addr.IsUnspecified() ||
		addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsMulticast() {
		return true
	}

	if addr.Is4() {
		octets := addr.As4()
		if octets[0] == 100 && octets[1] >= 64 && octets[1] <= 127 {
			return true
		}
	}

	return false
}

func isUnsafeRemoteImageHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" ||
		host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.Contains(host, "%") {
		return true
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return isUnsafeRemoteImageAddr(addr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false
	}
	for _, ip := range addrs {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok || isUnsafeRemoteImageAddr(addr) {
			return true
		}
	}

	return false
}

func validateRemoteImageURL(source string) (*neturl.URL, error) {
	instance, err := neturl.Parse(strings.TrimSpace(source))
	if err != nil || instance == nil || instance.Host == "" {
		return nil, fmt.Errorf("invalid image url")
	}
	if instance.Scheme != "http" && instance.Scheme != "https" {
		return nil, fmt.Errorf("unsupported image url scheme")
	}
	if instance.User != nil {
		return nil, fmt.Errorf("image url credentials are not allowed")
	}
	if isUnsafeRemoteImageHost(instance.Hostname()) {
		return nil, fmt.Errorf("local or private image urls are not allowed")
	}

	return instance, nil
}

func remoteImageDialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if isUnsafeRemoteImageHost(host) {
		return nil, fmt.Errorf("local or private image urls are not allowed")
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("image url host has no ip addresses")
	}

	target := ""
	for _, ip := range addrs {
		addr, ok := netip.AddrFromSlice(ip.IP)
		if !ok || isUnsafeRemoteImageAddr(addr) {
			return nil, fmt.Errorf("local or private image urls are not allowed")
		}

		addr = addr.Unmap()
		if target != "" {
			continue
		}
		if network == "tcp4" && !addr.Is4() {
			continue
		}
		if network == "tcp6" && !addr.Is6() {
			continue
		}
		target = net.JoinHostPort(addr.String(), port)
	}
	if target == "" {
		return nil, fmt.Errorf("image url host has no compatible ip addresses")
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, target)
}

func newRemoteImageHTTPClient() *http.Client {
	return &http.Client{
		Timeout: remoteImageFetchTimeout,
		Transport: &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: remoteImageDialContext,
		},
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			_, err := validateRemoteImageURL(req.URL.String())
			return err
		},
	}
}

func openRemoteImageResponse(source string) (*http.Response, error) {
	instance, err := validateRemoteImageURL(source)
	if err != nil {
		return nil, err
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

	data, _, err := readStoredImageSource(url)
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

	data, _, err := readStoredImageSource(url)
	if err != nil {
		return "", err
	}

	return Base64EncodeBytes(data), nil
}

func NormalizeImageForCapability(source string, capability ImageInputCapability) (*NormalizedImageInput, error) {
	mode := normalizeImageInputMode(capability.Mode)
	switch mode {
	case ImageInputModeURL:
		return normalizeImageForURL(source)
	case ImageInputModeVisionDataURL:
		return normalizeImageToVisionDataURL(source)
	case ImageInputModeInlineBase64:
		return normalizeImageForInlineBase64(source)
	default:
		return normalizeImageForURL(source)
	}
}

func normalizeImageInputMode(mode ImageInputMode) ImageInputMode {
	switch mode {
	case ImageInputModeURL, ImageInputModeVisionDataURL, ImageInputModeInlineBase64:
		return mode
	default:
		return ImageInputModeURL
	}
}

func normalizeImageForURL(source string) (*NormalizedImageInput, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("image source is empty")
	}

	if strings.HasPrefix(source, "data:image/") {
		return &NormalizedImageInput{
			Source: source,
			Mode:   ImageInputModeVisionDataURL,
		}, nil
	}

	if _, err := validateRemoteImageURL(source); err == nil {
		return &NormalizedImageInput{
			Source: source,
			Mode:   ImageInputModeURL,
		}, nil
	}

	if IsInternalAttachmentURL(source) {
		if publicURL := publicAttachmentImageURL(source); publicURL != "" {
			return &NormalizedImageInput{
				Source: publicURL,
				Mode:   ImageInputModeURL,
			}, nil
		}

		dataURL, err := NormalizeImageToVisionDataURL(source)
		if err != nil {
			return nil, err
		}
		return &NormalizedImageInput{
			Source: dataURL,
			Mode:   ImageInputModeVisionDataURL,
		}, nil
	}

	return &NormalizedImageInput{
		Source: source,
		Mode:   ImageInputModeURL,
	}, nil
}

func normalizeImageToVisionDataURL(source string) (*NormalizedImageInput, error) {
	dataURL, err := NormalizeImageToVisionDataURL(source)
	if err != nil {
		return nil, err
	}

	return &NormalizedImageInput{
		Source:   dataURL,
		MIMEType: "image/png",
		Mode:     ImageInputModeVisionDataURL,
	}, nil
}

func normalizeImageForInlineBase64(source string) (*NormalizedImageInput, error) {
	data, contentType, err := readStoredImageSource(source)
	if err != nil {
		return nil, err
	}

	rawBase64 := Base64EncodeBytes(data)
	normalizedType := normalizeContentType(contentType)
	if normalizedType == "" {
		normalizedType = NewImageContent(source).GetType()
	}
	if normalizedType == "" {
		normalizedType = "image/png"
	}

	return &NormalizedImageInput{
		Source:    fmt.Sprintf("data:%s;base64,%s", normalizedType, rawBase64),
		MIMEType:  normalizedType,
		RawBase64: rawBase64,
		Mode:      ImageInputModeInlineBase64,
	}, nil
}

func IsInternalAttachmentURL(source string) bool {
	if source == "" || strings.HasPrefix(source, "data:image/") {
		return false
	}

	_, ok := attachmentNameFromSource(source)
	return ok
}

func NormalizeInternalAttachmentImageURL(source string) (string, error) {
	if !IsInternalAttachmentURL(source) {
		return source, nil
	}
	if _, err := validateRemoteImageURL(source); err == nil {
		return source, nil
	}

	result, err := NormalizeImageForCapability(source, VisionDataURLImageInputCapability)
	if err != nil {
		return "", err
	}

	return result.Source, nil
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
