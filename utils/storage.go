package utils

import (
	"bytes"
	"chat/globals"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

const attachmentPrefix = "attachments"

var attachmentNamePattern = regexp.MustCompile(`attachments/([a-f0-9]{32}\.[A-Za-z0-9]+)`)

func AttachmentObjectKey(name string) string {
	return fmt.Sprintf("%s/%s", attachmentPrefix, name)
}

func AttachmentLocalPath(name string) string {
	return fmt.Sprintf("storage/%s/%s", attachmentPrefix, name)
}

func AttachmentPublicURL(name string) string {
	if strings.EqualFold(strings.TrimSpace(globals.StorageMode), "s3") &&
		strings.TrimSpace(globals.StorageS3PublicBaseURL) != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(globals.StorageS3PublicBaseURL, "/"), AttachmentObjectKey(name))
	}

	if strings.TrimSpace(globals.NotifyUrl) != "" {
		return fmt.Sprintf("%s/attachments/%s", strings.TrimSuffix(globals.NotifyUrl, "/"), name)
	}

	return fmt.Sprintf("/attachments/%s", name)
}

func ExtractAttachmentNames(data string) []string {
	matches := attachmentNamePattern.FindAllStringSubmatch(data, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		name := match[1]
		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}
		result = append(result, name)
	}

	return result
}

func s3StorageReady() bool {
	return strings.EqualFold(strings.TrimSpace(globals.StorageMode), "s3") &&
		strings.TrimSpace(globals.StorageS3Bucket) != "" &&
		strings.TrimSpace(globals.StorageS3Region) != "" &&
		strings.TrimSpace(globals.StorageS3AccessKey) != "" &&
		strings.TrimSpace(globals.StorageS3SecretKey) != ""
}

func newS3Client(ctx context.Context) (*s3.Client, error) {
	if !s3StorageReady() {
		return nil, fmt.Errorf("s3 storage is not configured")
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(globals.StorageS3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			globals.StorageS3AccessKey,
			globals.StorageS3SecretKey,
			"",
		)),
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = globals.StorageS3ForcePathStyle
		if strings.TrimSpace(globals.StorageS3Endpoint) != "" {
			o.BaseEndpoint = aws.String(globals.StorageS3Endpoint)
		}
	}), nil
}

func normalizeContentType(contentType string) string {
	return strings.TrimSpace(strings.Split(contentType, ";")[0])
}

func extensionFromContentType(contentType string) string {
	switch normalizeContentType(contentType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	case "image/svg+xml":
		return ".svg"
	}

	exts, err := mime.ExtensionsByType(normalizeContentType(contentType))
	if err != nil || len(exts) == 0 {
		return ""
	}

	return strings.ToLower(exts[0])
}

func attachmentNameForSource(source string, contentType string) string {
	ext := extensionFromContentType(contentType)
	if ext == "" {
		if instance, err := neturl.Parse(source); err == nil {
			ext = strings.ToLower(path.Ext(instance.Path))
		} else {
			ext = strings.ToLower(path.Ext(source))
		}
	}

	if ext == "" {
		ext = ".bin"
	}

	return Md5Encrypt(source) + ext
}

func readStoredImageSource(source string) ([]byte, string, error) {
	if strings.HasPrefix(source, "data:image/") {
		parts := SafeSplit(source, ",", 2)
		if len(parts) < 2 || parts[1] == "" {
			return nil, "", fmt.Errorf("invalid base64 image")
		}

		contentType := normalizeContentType(strings.TrimPrefix(SafeSplit(parts[0], ";", 2)[0], "data:"))
		decoded, err := Base64Decode(parts[1])
		if err != nil {
			return nil, "", err
		}

		return decoded, contentType, nil
	}

	res, err := http.Get(source)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		return nil, "", fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}

	contentType := normalizeContentType(res.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = normalizeContentType(http.DetectContentType(data))
	}

	return data, contentType, nil
}

func writeAttachmentLocal(name string, data []byte) error {
	FileDirSafe(AttachmentLocalPath(name))
	return os.WriteFile(AttachmentLocalPath(name), data, 0o644)
}

func writeAttachmentS3(ctx context.Context, name string, data []byte, contentType string) error {
	client, err := newS3Client(ctx)
	if err != nil {
		return err
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(globals.StorageS3Bucket),
		Key:         aws.String(AttachmentObjectKey(name)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(normalizeContentType(contentType)),
	})
	return err
}

func readAttachmentS3(ctx context.Context, name string) ([]byte, string, error) {
	client, err := newS3Client(ctx)
	if err != nil {
		return nil, "", err
	}

	object, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(globals.StorageS3Bucket),
		Key:    aws.String(AttachmentObjectKey(name)),
	})
	if err != nil {
		return nil, "", err
	}
	defer object.Body.Close()

	data, err := io.ReadAll(object.Body)
	if err != nil {
		return nil, "", err
	}

	return data, normalizeContentType(aws.ToString(object.ContentType)), nil
}

func deleteAttachmentS3(ctx context.Context, name string) error {
	client, err := newS3Client(ctx)
	if err != nil {
		return err
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(globals.StorageS3Bucket),
		Key:    aws.String(AttachmentObjectKey(name)),
	})
	return err
}

func StoreImage(source string) string {
	if !globals.AcceptImageStore {
		return source
	}

	data, contentType, err := readStoredImageSource(source)
	if err != nil {
		globals.Warn(fmt.Sprintf("[utils] load image source error: %s", err.Error()))
		return source
	}

	name := attachmentNameForSource(source, contentType)
	if strings.EqualFold(strings.TrimSpace(globals.StorageMode), "s3") {
		if err := writeAttachmentS3(context.Background(), name, data, contentType); err != nil {
			globals.Warn(fmt.Sprintf("[utils] upload image error: %s", err.Error()))
			return source
		}
	} else if err := writeAttachmentLocal(name, data); err != nil {
		globals.Warn(fmt.Sprintf("[utils] save image error: %s", err.Error()))
		return source
	}

	return AttachmentPublicURL(name)
}

func ServeStoredAttachment(c *gin.Context, name string) {
	localPath := AttachmentLocalPath(name)
	if IsFileExist(localPath) {
		c.File(localPath)
		return
	}

	if strings.EqualFold(strings.TrimSpace(globals.StorageMode), "s3") &&
		strings.TrimSpace(globals.StorageS3PublicBaseURL) != "" {
		c.Redirect(http.StatusTemporaryRedirect, AttachmentPublicURL(name))
		return
	}

	if s3StorageReady() {
		data, contentType, err := readAttachmentS3(c.Request.Context(), name)
		if err != nil {
			globals.Warn(fmt.Sprintf("[utils] read s3 attachment error: %s", err.Error()))
			c.Status(http.StatusNotFound)
			return
		}

		if contentType == "" {
			contentType = normalizeContentType(http.DetectContentType(data))
		}

		c.Data(http.StatusOK, contentType, data)
		return
	}

	c.Status(http.StatusNotFound)
}

func DeleteStoredAttachment(name string) error {
	var result error

	localPath := AttachmentLocalPath(name)
	if IsFileExist(localPath) {
		if err := DeleteFile(localPath); err != nil && !os.IsNotExist(err) {
			result = err
		}
	}

	if s3StorageReady() {
		if err := deleteAttachmentS3(context.Background(), name); err != nil {
			if result == nil {
				result = err
			}
		}
	}

	return result
}
