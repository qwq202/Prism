package utils

import (
	"bytes"
	"chat/globals"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

const attachmentPrefix = "attachments"

var (
	attachmentNamePattern     = regexp.MustCompile(`attachments/([a-f0-9]{32}\.[A-Za-z0-9]+)`)
	storedAttachmentNameRegex = regexp.MustCompile(`^[a-f0-9]{32}\.[A-Za-z0-9]+$`)
)

type StorageS3Config struct {
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	PublicBaseURL  string
	ForcePathStyle bool
}

type StorageR2Config struct {
	AccountID     string
	Jurisdiction  string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string
}

type StorageTestConfig struct {
	Mode    string
	Backend string
	S3      StorageS3Config
	R2      StorageR2Config
}

type StoredAttachmentInfo struct {
	Name        string
	Size        int64
	UpdatedAt   time.Time
	StorageMode string
	PublicURL   string
}

type storageClientConfig struct {
	Mode           string
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	PublicBaseURL  string
	ForcePathStyle bool
	Backend        string
}

func AttachmentObjectKey(name string) string {
	return fmt.Sprintf("%s/%s", attachmentPrefix, name)
}

func AttachmentLocalPath(name string) string {
	return fmt.Sprintf("storage/%s/%s", attachmentPrefix, name)
}

func AttachmentPublicURL(name string) string {
	if publicBaseURL := getStoragePublicBaseURL(); publicBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(publicBaseURL, "/"), AttachmentObjectKey(name))
	}

	if strings.TrimSpace(globals.NotifyUrl) != "" {
		return fmt.Sprintf("%s/attachments/%s", strings.TrimSuffix(globals.NotifyUrl, "/"), name)
	}

	return fmt.Sprintf("/attachments/%s", name)
}

func validateStoredAttachmentName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("attachment name is required")
	}
	if !storedAttachmentNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid attachment name")
	}
	return name, nil
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
	return storageReadyWithConfig(currentStorageConfig())
}

func r2StorageReady() bool {
	current := currentStorageConfig()
	return strings.EqualFold(strings.TrimSpace(current.Mode), "r2") &&
		storageReadyWithConfig(current)
}

func getStoragePublicBaseURL() string {
	return strings.TrimSpace(currentStorageConfig().PublicBaseURL)
}

func getR2Endpoint() string {
	accountID := strings.TrimSpace(globals.StorageR2AccountID)
	if accountID == "" {
		return ""
	}

	jurisdiction := strings.TrimSpace(globals.StorageR2Jurisdiction)
	if jurisdiction == "" {
		return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}

	return fmt.Sprintf("https://%s.%s.r2.cloudflarestorage.com", accountID, jurisdiction)
}

func currentStorageConfig() storageClientConfig {
	mode := strings.ToLower(strings.TrimSpace(globals.StorageMode))
	switch mode {
	case "s3":
		return storageClientConfig{
			Mode:           "s3",
			Endpoint:       strings.TrimSpace(globals.StorageS3Endpoint),
			Region:         strings.TrimSpace(globals.StorageS3Region),
			Bucket:         strings.TrimSpace(globals.StorageS3Bucket),
			AccessKey:      strings.TrimSpace(globals.StorageS3AccessKey),
			SecretKey:      strings.TrimSpace(globals.StorageS3SecretKey),
			PublicBaseURL:  strings.TrimSpace(globals.StorageS3PublicBaseURL),
			ForcePathStyle: globals.StorageS3ForcePathStyle,
			Backend:        strings.TrimSpace(globals.NotifyUrl),
		}
	case "r2":
		return storageClientConfig{
			Mode:           "r2",
			Endpoint:       strings.TrimSpace(getR2Endpoint()),
			Region:         "auto",
			Bucket:         strings.TrimSpace(globals.StorageR2Bucket),
			AccessKey:      strings.TrimSpace(globals.StorageR2AccessKey),
			SecretKey:      strings.TrimSpace(globals.StorageR2SecretKey),
			PublicBaseURL:  strings.TrimSpace(globals.StorageR2PublicBaseURL),
			ForcePathStyle: true,
			Backend:        strings.TrimSpace(globals.NotifyUrl),
		}
	default:
		return storageClientConfig{
			Mode:    "local",
			Backend: strings.TrimSpace(globals.NotifyUrl),
		}
	}
}

func buildStorageTestConfig(config StorageTestConfig) storageClientConfig {
	mode := strings.ToLower(strings.TrimSpace(config.Mode))
	switch mode {
	case "s3":
		return storageClientConfig{
			Mode:           "s3",
			Endpoint:       strings.TrimSuffix(strings.TrimSpace(config.S3.Endpoint), "/"),
			Region:         strings.TrimSpace(config.S3.Region),
			Bucket:         strings.TrimSpace(config.S3.Bucket),
			AccessKey:      strings.TrimSpace(config.S3.AccessKey),
			SecretKey:      strings.TrimSpace(config.S3.SecretKey),
			PublicBaseURL:  strings.TrimSuffix(strings.TrimSpace(config.S3.PublicBaseURL), "/"),
			ForcePathStyle: config.S3.ForcePathStyle,
			Backend:        strings.TrimSuffix(strings.TrimSpace(config.Backend), "/"),
		}
	case "r2":
		accountID := strings.TrimSpace(config.R2.AccountID)
		jurisdiction := strings.TrimSpace(config.R2.Jurisdiction)
		endpoint := ""
		if accountID != "" {
			if jurisdiction == "" {
				endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
			} else {
				endpoint = fmt.Sprintf("https://%s.%s.r2.cloudflarestorage.com", accountID, jurisdiction)
			}
		}
		return storageClientConfig{
			Mode:           "r2",
			Endpoint:       endpoint,
			Region:         "auto",
			Bucket:         strings.TrimSpace(config.R2.Bucket),
			AccessKey:      strings.TrimSpace(config.R2.AccessKey),
			SecretKey:      strings.TrimSpace(config.R2.SecretKey),
			PublicBaseURL:  strings.TrimSuffix(strings.TrimSpace(config.R2.PublicBaseURL), "/"),
			ForcePathStyle: true,
			Backend:        strings.TrimSuffix(strings.TrimSpace(config.Backend), "/"),
		}
	default:
		return storageClientConfig{
			Mode:    "local",
			Backend: strings.TrimSuffix(strings.TrimSpace(config.Backend), "/"),
		}
	}
}

func isBlockedPublicStorageHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}

	blockedSuffixes := []string{
		"r2.cloudflarestorage.com",
	}

	for _, suffix := range blockedSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}

	return false
}

func storageReadyWithConfig(config storageClientConfig) bool {
	switch strings.ToLower(strings.TrimSpace(config.Mode)) {
	case "s3", "r2":
		return strings.TrimSpace(config.Bucket) != "" &&
			strings.TrimSpace(config.Region) != "" &&
			strings.TrimSpace(config.AccessKey) != "" &&
			strings.TrimSpace(config.SecretKey) != ""
	default:
		return true
	}
}

func newS3ClientWithConfig(ctx context.Context, config storageClientConfig) (*s3.Client, error) {
	if !storageReadyWithConfig(config) || !Contains(config.Mode, []string{"s3", "r2"}) {
		return nil, fmt.Errorf("%s storage is not configured", config.Mode)
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(config.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AccessKey,
			config.SecretKey,
			"",
		)),
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = config.ForcePathStyle
		if config.Endpoint != "" {
			o.BaseEndpoint = aws.String(config.Endpoint)
		}
	}), nil
}

func newS3Client(ctx context.Context) (*s3.Client, error) {
	return newS3ClientWithConfig(ctx, currentStorageConfig())
}

func storageBucket() string {
	return currentStorageConfig().Bucket
}

func storageModeReady() bool {
	current := currentStorageConfig()
	return Contains(current.Mode, []string{"s3", "r2"}) && storageReadyWithConfig(current)
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

func attachmentNameForUpload(filename string, data []byte, contentType string) string {
	ext := extensionFromContentType(contentType)
	if ext == "" {
		ext = strings.ToLower(path.Ext(filename))
	}
	if ext == "" {
		ext = ".bin"
	}

	sum := md5.Sum(data)
	return fmt.Sprintf("%x%s", sum, ext)
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
		if int64(len(decoded)) > maxRemoteImageBytes {
			return nil, "", remoteImageSizeError(maxRemoteImageBytes)
		}

		return decoded, contentType, nil
	}

	data, contentType, err := readRemoteImageBytes(source, maxRemoteImageBytes)
	if err != nil {
		return nil, "", err
	}

	return data, contentType, nil
}

func writeAttachmentLocal(name string, data []byte) error {
	FileDirSafe(AttachmentLocalPath(name))
	return os.WriteFile(AttachmentLocalPath(name), data, 0o644)
}

func writeAttachmentS3(ctx context.Context, name string, data []byte, contentType string) error {
	config := currentStorageConfig()
	return writeAttachmentS3WithConfig(ctx, config, name, data, contentType)
}

func writeAttachmentS3WithConfig(ctx context.Context, config storageClientConfig, name string, data []byte, contentType string) error {
	client, err := newS3ClientWithConfig(ctx, config)
	if err != nil {
		return err
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(config.Bucket),
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
		Bucket: aws.String(storageBucket()),
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
	return deleteAttachmentS3WithConfig(ctx, currentStorageConfig(), name)
}

func deleteAttachmentS3WithConfig(ctx context.Context, config storageClientConfig, name string) error {
	client, err := newS3ClientWithConfig(ctx, config)
	if err != nil {
		return err
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(config.Bucket),
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
	if storageModeReady() && !strings.EqualFold(strings.TrimSpace(globals.StorageMode), "local") {
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

func StoreAttachmentData(filename string, data []byte, contentType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("attachment data is empty")
	}

	contentType = normalizeContentType(contentType)
	if contentType == "" {
		contentType = normalizeContentType(http.DetectContentType(data))
	}

	name := attachmentNameForUpload(filename, data, contentType)
	if storageModeReady() && !strings.EqualFold(strings.TrimSpace(globals.StorageMode), "local") {
		if err := writeAttachmentS3(context.Background(), name, data, contentType); err != nil {
			return "", err
		}
	} else if err := writeAttachmentLocal(name, data); err != nil {
		return "", err
	}

	return AttachmentPublicURL(name), nil
}

func ServeStoredAttachment(c *gin.Context, name string) {
	name, err := validateStoredAttachmentName(name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	localPath := AttachmentLocalPath(name)
	if IsFileExist(localPath) {
		c.File(localPath)
		return
	}

	if publicBaseURL := getStoragePublicBaseURL(); publicBaseURL != "" {
		c.Redirect(http.StatusTemporaryRedirect, AttachmentPublicURL(name))
		return
	}

	if storageModeReady() {
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

	name, err := validateStoredAttachmentName(name)
	if err != nil {
		return err
	}

	localPath := AttachmentLocalPath(name)
	if IsFileExist(localPath) {
		if err := DeleteFile(localPath); err != nil && !os.IsNotExist(err) {
			result = err
		}
	}

	if storageModeReady() {
		if err := deleteAttachmentS3(context.Background(), name); err != nil {
			if result == nil {
				result = err
			}
		}
	}

	return result
}

func DeleteConfiguredStoredAttachment(name string) error {
	var err error
	name, err = validateStoredAttachmentName(name)
	if err != nil {
		return err
	}

	current := currentStorageConfig()
	if Contains(current.Mode, []string{"s3", "r2"}) && storageReadyWithConfig(current) {
		return deleteAttachmentS3(context.Background(), name)
	}

	if err := DeleteFile(AttachmentLocalPath(name)); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func ListStoredAttachmentNames() ([]string, error) {
	seen := map[string]struct{}{}
	result := make([]string, 0)

	appendName := func(name string) {
		name = strings.TrimSpace(name)
		if _, err := validateStoredAttachmentName(name); err != nil {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}

	localDir := filepath.Dir(AttachmentLocalPath("placeholder"))
	if entries, err := os.ReadDir(localDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			appendName(entry.Name())
		}
	}

	if storageModeReady() {
		ctx := context.Background()
		client, err := newS3Client(ctx)
		if err != nil {
			return nil, err
		}

		paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
			Bucket: aws.String(storageBucket()),
			Prefix: aws.String(attachmentPrefix + "/"),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, object := range page.Contents {
				key := strings.TrimSpace(aws.ToString(object.Key))
				if key == "" || !strings.HasPrefix(key, attachmentPrefix+"/") {
					continue
				}
				appendName(strings.TrimPrefix(key, attachmentPrefix+"/"))
			}
		}
	}

	return result, nil
}

func ListConfiguredStoredAttachments() ([]StoredAttachmentInfo, error) {
	current := currentStorageConfig()
	result := make([]StoredAttachmentInfo, 0)

	appendAttachment := func(item StoredAttachmentInfo) {
		item.Name = strings.TrimSpace(item.Name)
		if _, err := validateStoredAttachmentName(item.Name); err != nil {
			return
		}
		item.StorageMode = current.Mode
		item.PublicURL = AttachmentPublicURL(item.Name)
		result = append(result, item)
	}

	if Contains(current.Mode, []string{"s3", "r2"}) && storageReadyWithConfig(current) {
		ctx := context.Background()
		client, err := newS3Client(ctx)
		if err != nil {
			return nil, err
		}

		paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
			Bucket: aws.String(storageBucket()),
			Prefix: aws.String(attachmentPrefix + "/"),
		})

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, object := range page.Contents {
				key := strings.TrimSpace(aws.ToString(object.Key))
				if key == "" || !strings.HasPrefix(key, attachmentPrefix+"/") {
					continue
				}

				appendAttachment(StoredAttachmentInfo{
					Name:      strings.TrimPrefix(key, attachmentPrefix+"/"),
					Size:      aws.ToInt64(object.Size),
					UpdatedAt: aws.ToTime(object.LastModified),
				})
			}
		}
	} else {
		localDir := filepath.Dir(AttachmentLocalPath("placeholder"))
		entries, err := os.ReadDir(localDir)
		if err != nil {
			if os.IsNotExist(err) {
				return result, nil
			}
			return nil, err
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			appendAttachment(StoredAttachmentInfo{
				Name:      entry.Name(),
				Size:      info.Size(),
				UpdatedAt: info.ModTime(),
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result, nil
}

func TestStorageConnection(config StorageTestConfig) error {
	current := buildStorageTestConfig(config)
	if strings.TrimSpace(current.PublicBaseURL) == "" && strings.TrimSpace(current.Backend) == "" {
		return fmt.Errorf("public base url or backend domain is required")
	}

	if strings.TrimSpace(current.PublicBaseURL) != "" {
		instance, err := neturl.Parse(current.PublicBaseURL)
		if err != nil {
			return fmt.Errorf("invalid public base url: %w", err)
		}

		if isBlockedPublicStorageHost(instance.Hostname()) {
			return fmt.Errorf("public base url must be a real public file url such as r2.dev or a custom domain, not the object storage api endpoint")
		}
	}

	content := []byte(fmt.Sprintf("storage-test-%d", time.Now().UnixNano()))
	name := attachmentNameForUpload("storage-test.txt", content, "text/plain")

	switch current.Mode {
	case "local":
		if err := writeAttachmentLocal(name, content); err != nil {
			return err
		}
		if err := DeleteFile(AttachmentLocalPath(name)); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "s3", "r2":
		if err := writeAttachmentS3WithConfig(context.Background(), current, name, content, "text/plain"); err != nil {
			return err
		}
		if err := deleteAttachmentS3WithConfig(context.Background(), current, name); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported storage mode")
	}
}
