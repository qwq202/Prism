package channel

import (
	"chat/globals"
	"chat/utils"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type ApiInfo struct {
	Title        string   `json:"title"`
	Logo         string   `json:"logo"`
	File         string   `json:"file"`
	Docs         string   `json:"docs"`
	Announcement string   `json:"announcement"`
	BuyLink      string   `json:"buy_link"`
	Contact      string   `json:"contact"`
	Footer       string   `json:"footer"`
	AuthFooter   bool     `json:"auth_footer"`
	Mail         bool     `json:"mail"`
	Article      []string `json:"article"`
	Generation   []string `json:"generation"`
	RelayPlan    bool     `json:"relay_plan"`
	WebSearch    bool     `json:"web_search"`
	HasTaskModel bool     `json:"has_task_model"`
}

type generalState struct {
	Title       string `json:"title" mapstructure:"title"`
	Logo        string `json:"logo" mapstructure:"logo"`
	Backend     string `json:"backend" mapstructure:"backend"`
	File        string `json:"file" mapstructure:"file"`
	Docs        string `json:"docs" mapstructure:"docs"`
	PWAManifest string `json:"pwa_manifest" mapstructure:"pwamanifest"`
	DebugMode   bool   `json:"debug_mode" mapstructure:"debugmode"`
}

type siteState struct {
	CloseRegister bool    `json:"close_register" mapstructure:"closeregister"`
	CloseRelay    bool    `json:"close_relay" mapstructure:"closerelay"`
	RelayPlan     bool    `json:"relay_plan" mapstructure:"relayplan"`
	Quota         float64 `json:"quota" mapstructure:"quota"`
	BuyLink       string  `json:"buy_link" mapstructure:"buylink"`
	Announcement  string  `json:"announcement" mapstructure:"announcement"`
	Contact       string  `json:"contact" mapstructure:"contact"`
	Footer        string  `json:"footer" mapstructure:"footer"`
	AuthFooter    bool    `json:"auth_footer" mapstructure:"authfooter"`
}

type whiteList struct {
	Enabled   bool     `json:"enabled" mapstructure:"enabled"`
	Custom    string   `json:"custom" mapstructure:"custom"`
	WhiteList []string `json:"white_list" mapstructure:"whitelist"`
}

type mailState struct {
	Host      string    `json:"host" mapstructure:"host"`
	Protocol  bool      `json:"protocol" mapstructure:"protocol"`
	Port      int       `json:"port" mapstructure:"port"`
	Username  string    `json:"username" mapstructure:"username"`
	Password  string    `json:"password" mapstructure:"password"`
	From      string    `json:"from" mapstructure:"from"`
	WhiteList whiteList `json:"white_list" mapstructure:"whitelist"`
}

type SearchState struct {
	ApiKey     string `json:"api_key" mapstructure:"apikey"`
	Crop       bool   `json:"crop" mapstructure:"crop"`
	CropLen    int    `json:"crop_len" mapstructure:"croplen"`
	MaxResults int    `json:"max_results" mapstructure:"maxresults"`
	Topic      string `json:"topic" mapstructure:"topic"`
	Depth      string `json:"depth" mapstructure:"depth"`
}

type taskState struct {
	Model string `json:"model" mapstructure:"model"`
}

type s3StorageState struct {
	Endpoint       string `json:"endpoint" mapstructure:"endpoint"`
	Region         string `json:"region" mapstructure:"region"`
	Bucket         string `json:"bucket" mapstructure:"bucket"`
	AccessKey      string `json:"access_key" mapstructure:"access_key"`
	SecretKey      string `json:"secret_key" mapstructure:"secret_key"`
	PublicBaseURL  string `json:"public_base_url" mapstructure:"public_base_url"`
	ForcePathStyle bool   `json:"force_path_style" mapstructure:"force_path_style"`
}

type commonState struct {
	Article     []string       `json:"article" mapstructure:"article"`
	Generation  []string       `json:"generation" mapstructure:"generation"`
	Cache       []string       `json:"cache" mapstructure:"cache"`
	Expire      int64          `json:"expire" mapstructure:"expire"`
	Size        int64          `json:"size" mapstructure:"size"`
	ImageStore  bool           `json:"image_store" mapstructure:"imagestore"`
	PromptStore bool           `json:"prompt_store" mapstructure:"promptstore"`
	StorageMode string         `json:"storage_mode" mapstructure:"storage_mode"`
	S3          s3StorageState `json:"s3" mapstructure:"s3"`
}

type SystemConfig struct {
	General generalState `json:"general" mapstructure:"general"`
	Site    siteState    `json:"site" mapstructure:"site"`
	Mail    mailState    `json:"mail" mapstructure:"mail"`
	Search  SearchState  `json:"search" mapstructure:"search"`
	Task    taskState    `json:"task" mapstructure:"task"`
	Common  commonState  `json:"common" mapstructure:"common"`
}

func NewSystemConfig() *SystemConfig {
	conf := &SystemConfig{}
	if err := viper.UnmarshalKey("system", conf); err != nil {
		panic(err)
	}

	conf.Load()
	return conf
}

func (c *SystemConfig) Load() {
	globals.NotifyUrl = c.GetBackend()
	globals.DebugMode = c.General.DebugMode

	globals.CloseRegistration = c.Site.CloseRegister
	globals.CloseRelay = c.Site.CloseRelay

	globals.ArticlePermissionGroup = c.Common.Article
	globals.GenerationPermissionGroup = c.Common.Generation
	globals.CacheAcceptedModels = c.Common.Cache

	globals.CacheAcceptedExpire = c.GetCacheAcceptedExpire()
	globals.CacheAcceptedSize = c.GetCacheAcceptedSize()
	globals.StorageMode = c.GetStorageMode()
	globals.StorageS3Endpoint = c.GetStorageS3Endpoint()
	globals.StorageS3Region = c.GetStorageS3Region()
	globals.StorageS3Bucket = c.GetStorageS3Bucket()
	globals.StorageS3AccessKey = c.GetStorageS3AccessKey()
	globals.StorageS3SecretKey = c.GetStorageS3SecretKey()
	globals.StorageS3PublicBaseURL = c.GetStorageS3PublicBaseURL()
	globals.StorageS3ForcePathStyle = c.Common.S3.ForcePathStyle
	globals.AcceptImageStore = c.AcceptImageStore()

	globals.AcceptPromptStore = c.Common.PromptStore

	if c.General.PWAManifest == "" {
		c.General.PWAManifest = utils.ReadPWAManifest()
	}

	globals.SearchApiKey = strings.TrimSpace(c.Search.ApiKey)
	globals.SearchCrop = c.Search.Crop
	globals.SearchCropLength = c.GetSearchCropLength()
	globals.SearchMaxResults = c.GetSearchMaxResults()
	globals.SearchTopic = c.GetSearchTopic()
	globals.SearchDepth = c.GetSearchDepth()
	globals.TaskModel = strings.TrimSpace(c.Task.Model)
}

func (c *SystemConfig) SaveConfig() error {
	return utils.SaveConfig("system", c)
}

func (c *SystemConfig) AsInfo() ApiInfo {
	return ApiInfo{
		Title:        c.General.Title,
		Logo:         c.General.Logo,
		File:         c.General.File,
		Docs:         c.General.Docs,
		Announcement: c.Site.Announcement,
		Contact:      c.Site.Contact,
		Footer:       c.Site.Footer,
		AuthFooter:   c.Site.AuthFooter,
		BuyLink:      c.Site.BuyLink,
		Mail:         c.IsMailValid(),
		Article:      c.Common.Article,
		Generation:   c.Common.Generation,
		RelayPlan:    c.Site.RelayPlan,
		WebSearch:    strings.TrimSpace(globals.SearchApiKey) != "",
		HasTaskModel: strings.TrimSpace(globals.TaskModel) != "",
	}
}

func (c *SystemConfig) UpdateConfig(data *SystemConfig) error {
	c.General = data.General
	c.Site = data.Site
	c.Mail = data.Mail
	c.Search = data.Search
	c.Task = data.Task
	c.Common = data.Common

	utils.ApplySeo(c.General.Title, c.General.Logo)
	utils.ApplyPWAManifest(c.General.PWAManifest)

	if err := c.SaveConfig(); err != nil {
		return err
	}

	c.Load()
	return nil
}

func (c *SystemConfig) GetInitialQuota() float64 {
	return c.Site.Quota
}

func (c *SystemConfig) GetBackend() string {
	return strings.TrimSuffix(c.General.Backend, "/")
}

func (c *SystemConfig) GetMail() *utils.SmtpPoster {
	return utils.NewSmtpPoster(
		c.Mail.Host,
		c.Mail.Protocol,
		c.Mail.Port,
		c.Mail.Username,
		c.Mail.Password,
		c.Mail.From,
	)
}

func (c *SystemConfig) IsMailValid() bool {
	return c.GetMail().Valid()
}

func (c *SystemConfig) GetMailSuffix() []string {
	if c.Mail.WhiteList.Enabled {
		return c.Mail.WhiteList.WhiteList
	}

	return []string{}
}

func (c *SystemConfig) IsValidMailSuffix(suffix string) bool {
	if c.Mail.WhiteList.Enabled {
		return utils.Contains(suffix, c.Mail.WhiteList.WhiteList)
	}

	return true
}

func (c *SystemConfig) IsValidMail(email string) error {
	segment := strings.Split(email, "@")
	if len(segment) != 2 {
		return fmt.Errorf("invalid email format")
	}

	if suffix := segment[1]; !c.IsValidMailSuffix(suffix) {
		return fmt.Errorf("email suffix @%s is not allowed to register", suffix)
	}

	return nil
}

func (c *SystemConfig) SendVerifyMail(email string, code string) error {
	type Temp struct {
		Title string `json:"title"`
		Logo  string `json:"logo"`
		Code  string `json:"code"`
	}

	return c.GetMail().RenderMail(
		"code.html",
		Temp{Title: c.GetAppName(), Logo: c.GetAppLogo(), Code: code},
		email,
		fmt.Sprintf("%s | OTP Verification", c.GetAppName()),
	)
}

func (c *SystemConfig) GetSearchCropLength() int {
	if c.Search.CropLen <= 0 {
		return 1000
	}

	return c.Search.CropLen
}

func (c *SystemConfig) GetSearchMaxResults() int {
	if c.Search.MaxResults <= 0 {
		return 5
	}

	if c.Search.MaxResults > 20 {
		return 20
	}

	return c.Search.MaxResults
}

func (c *SystemConfig) GetSearchTopic() string {
	topic := strings.TrimSpace(c.Search.Topic)
	switch topic {
	case "general", "news", "finance":
		return topic
	default:
		return "general"
	}
}

func (c *SystemConfig) GetSearchDepth() string {
	depth := strings.TrimSpace(c.Search.Depth)
	switch depth {
	case "basic", "advanced", "fast", "ultra-fast":
		return depth
	default:
		return "basic"
	}
}

func (c *SystemConfig) GetAppName() string {
	title := strings.TrimSpace(c.General.Title)
	if len(title) == 0 {
		return "CoAI.Dev"
	}

	return title
}

func (c *SystemConfig) GetAppLogo() string {
	logo := strings.TrimSpace(c.General.Logo)
	if len(logo) == 0 {
		return "https://chatnio.net/favicon.ico"
	}

	return logo
}

func (c *SystemConfig) GetCacheAcceptedModels() []string {
	return c.Common.Cache
}

func (c *SystemConfig) GetCacheAcceptedExpire() int64 {
	if c.Common.Expire <= 0 {
		// default 1 hour
		return 3600
	}

	return c.Common.Expire
}

func (c *SystemConfig) GetCacheAcceptedSize() int64 {
	if c.Common.Size < 1 {
		return 1
	}

	return c.Common.Size
}

func (c *SystemConfig) GetStorageMode() string {
	if strings.EqualFold(strings.TrimSpace(c.Common.StorageMode), "s3") {
		return "s3"
	}

	return "local"
}

func (c *SystemConfig) GetStorageS3Endpoint() string {
	return strings.TrimSuffix(strings.TrimSpace(c.Common.S3.Endpoint), "/")
}

func (c *SystemConfig) GetStorageS3Region() string {
	return strings.TrimSpace(c.Common.S3.Region)
}

func (c *SystemConfig) GetStorageS3Bucket() string {
	return strings.TrimSpace(c.Common.S3.Bucket)
}

func (c *SystemConfig) GetStorageS3AccessKey() string {
	return strings.TrimSpace(c.Common.S3.AccessKey)
}

func (c *SystemConfig) GetStorageS3SecretKey() string {
	return strings.TrimSpace(c.Common.S3.SecretKey)
}

func (c *SystemConfig) GetStorageS3PublicBaseURL() string {
	return strings.TrimSuffix(strings.TrimSpace(c.Common.S3.PublicBaseURL), "/")
}

func (c *SystemConfig) IsS3StorageConfigured() bool {
	return c.GetStorageS3Bucket() != "" &&
		c.GetStorageS3Region() != "" &&
		c.GetStorageS3AccessKey() != "" &&
		c.GetStorageS3SecretKey() != ""
}

func (c *SystemConfig) AcceptImageStore() bool {
	if !c.Common.ImageStore {
		return false
	}

	if c.GetStorageMode() == "s3" {
		if !c.IsS3StorageConfigured() {
			return false
		}

		return c.GetStorageS3PublicBaseURL() != "" || len(strings.TrimSpace(globals.NotifyUrl)) > 0
	}

	return len(strings.TrimSpace(globals.NotifyUrl)) > 0
}

func (c *SystemConfig) SupportRelayPlan() bool {
	return c.Site.RelayPlan
}
