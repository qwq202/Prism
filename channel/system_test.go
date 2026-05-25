package channel

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
)

func TestSystemConfigUnmarshalLoadsSnakeCaseKeys(t *testing.T) {
	loader := viper.New()
	loader.Set("system", map[string]any{
		"general": map[string]any{
			"backend":      " https://app.example.com/ ",
			"pwa_manifest": `{"name":"Prism"}`,
			"debug_mode":   true,
		},
		"site": map[string]any{
			"close_register": true,
			"buy_link":       "https://billing.example.com",
			"auth_footer":    true,
		},
		"search": map[string]any{
			"api_key":     "search-key",
			"crop_len":    800,
			"max_results": 7,
		},
		"common": map[string]any{
			"image_store":             true,
			"prompt_store":            true,
			"orphan_cleanup_enabled":  true,
			"orphan_cleanup_interval": 45,
			"storage_mode":            "s3",
			"s3": map[string]any{
				"endpoint":         "https://s3.example.com/",
				"region":           "us-east-1",
				"bucket":           "prism-files",
				"access_key":       "access-key",
				"secret_key":       "secret-key",
				"public_base_url":  "https://cdn.example.com/",
				"force_path_style": true,
			},
			"r2": map[string]any{
				"account_id":      "r2-account",
				"jurisdiction":    "EU",
				"bucket":          "prism-r2",
				"access_key":      "r2-access",
				"secret_key":      "r2-secret",
				"public_base_url": "https://pub.example.r2.dev/",
			},
		},
	})

	var conf SystemConfig
	if err := loader.UnmarshalKey("system", &conf, systemConfigDecoderOption); err != nil {
		t.Fatalf("unmarshal system config: %v", err)
	}

	if conf.General.PWAManifest != `{"name":"Prism"}` || !conf.General.DebugMode {
		t.Fatalf("expected snake_case general keys to load, got %#v", conf.General)
	}
	if !conf.Site.CloseRegister || conf.Site.BuyLink != "https://billing.example.com" || !conf.Site.AuthFooter {
		t.Fatalf("expected snake_case site keys to load, got %#v", conf.Site)
	}
	if conf.Search.ApiKey != "search-key" || conf.Search.CropLen != 800 || conf.Search.MaxResults != 7 {
		t.Fatalf("expected snake_case search keys to load, got %#v", conf.Search)
	}
	if !conf.Common.ImageStore || !conf.Common.PromptStore ||
		!conf.Common.OrphanCleanupEnabled || conf.Common.OrphanCleanupInterval != 45 ||
		conf.Common.StorageMode != "s3" {
		t.Fatalf("expected snake_case common keys to load, got %#v", conf.Common)
	}
	if conf.Common.S3.AccessKey != "access-key" ||
		conf.Common.S3.SecretKey != "secret-key" ||
		conf.Common.S3.PublicBaseURL != "https://cdn.example.com/" ||
		!conf.Common.S3.ForcePathStyle {
		t.Fatalf("expected snake_case s3 keys to load, got %#v", conf.Common.S3)
	}
	if conf.Common.R2.AccountID != "r2-account" ||
		conf.Common.R2.AccessKey != "r2-access" ||
		conf.Common.R2.PublicBaseURL != "https://pub.example.r2.dev/" {
		t.Fatalf("expected snake_case r2 keys to load, got %#v", conf.Common.R2)
	}
}

func TestSystemConfigNormalizeUsesRuntimeSafeValues(t *testing.T) {
	conf := &SystemConfig{
		General: generalState{
			Backend:  " https://app.example.com/ ",
			TimeZone: " Not/AZone ",
		},
		Search: SearchState{
			ApiKey:     " search-key ",
			CropLen:    0,
			MaxResults: 99,
			Topic:      "invalid-topic",
			Depth:      "invalid-depth",
		},
		Common: commonState{
			Expire:                0,
			Size:                  0,
			OrphanCleanupInterval: 2,
			StorageMode:           " R2 ",
			S3: s3StorageState{
				Endpoint:      " https://s3.example.com/ ",
				Region:        " us-east-1 ",
				Bucket:        " prism-files ",
				AccessKey:     " access-key ",
				SecretKey:     " secret-key ",
				PublicBaseURL: " https://cdn.example.com/ ",
			},
			R2: r2StorageState{
				AccountID:     " account-id ",
				Jurisdiction:  " EU ",
				Bucket:        " prism-r2 ",
				AccessKey:     " r2-access ",
				SecretKey:     " r2-secret ",
				PublicBaseURL: " https://r2.example.com/ ",
			},
		},
		Auth: authState{
			Passkey: passkeyState{
				RPDisplayName:           " Prism Keys ",
				RPID:                    " Example.COM ",
				UserVerification:        " required ",
				AuthenticatorAttachment: " cross-platform ",
				Origins:                 " https://one.example.com/ ,\n https://two.example.com/ \n",
			},
		},
	}

	conf.Normalize()

	if conf.General.Backend != "https://app.example.com" {
		t.Fatalf("expected backend to be trimmed, got %q", conf.General.Backend)
	}
	if conf.General.TimeZone != defaultSystemTimeZone {
		t.Fatalf("expected invalid timezone to default, got %q", conf.General.TimeZone)
	}
	if conf.Search.ApiKey != "search-key" ||
		conf.Search.CropLen != 1000 ||
		conf.Search.MaxResults != 20 ||
		conf.Search.Topic != "general" ||
		conf.Search.Depth != "basic" {
		t.Fatalf("unexpected normalized search config: %#v", conf.Search)
	}
	if conf.Common.Expire != 3600 || conf.Common.Size != 1 || conf.Common.OrphanCleanupInterval != 5 {
		t.Fatalf("unexpected normalized cache cleanup config: expire=%d size=%d interval=%d", conf.Common.Expire, conf.Common.Size, conf.Common.OrphanCleanupInterval)
	}
	if conf.Common.StorageMode != "r2" {
		t.Fatalf("expected storage mode to normalize to r2, got %q", conf.Common.StorageMode)
	}
	if conf.Common.S3.Endpoint != "https://s3.example.com" ||
		conf.Common.S3.Region != "us-east-1" ||
		conf.Common.S3.Bucket != "prism-files" ||
		conf.Common.S3.AccessKey != "access-key" ||
		conf.Common.S3.SecretKey != "secret-key" ||
		conf.Common.S3.PublicBaseURL != "https://cdn.example.com" {
		t.Fatalf("unexpected normalized s3 config: %#v", conf.Common.S3)
	}
	if conf.Common.R2.AccountID != "account-id" ||
		conf.Common.R2.Jurisdiction != "eu" ||
		conf.Common.R2.Bucket != "prism-r2" ||
		conf.Common.R2.AccessKey != "r2-access" ||
		conf.Common.R2.SecretKey != "r2-secret" ||
		conf.Common.R2.PublicBaseURL != "https://r2.example.com" {
		t.Fatalf("unexpected normalized r2 config: %#v", conf.Common.R2)
	}
	if conf.Auth.Passkey.RPDisplayName != "Prism Keys" ||
		conf.Auth.Passkey.RPID != "example.com" ||
		conf.Auth.Passkey.UserVerification != "required" ||
		conf.Auth.Passkey.AuthenticatorAttachment != "cross-platform" ||
		conf.Auth.Passkey.Origins != "https://one.example.com\nhttps://two.example.com" {
		t.Fatalf("unexpected normalized passkey config: %#v", conf.Auth.Passkey)
	}
}

func TestPasskeyGettersReturnValidBrowserEnums(t *testing.T) {
	conf := &SystemConfig{
		Auth: authState{
			Passkey: passkeyState{
				UserVerification:        " preferred ",
				AuthenticatorAttachment: " platform ",
			},
		},
	}
	if got := conf.GetPasskeyUserVerification(); got != "preferred" {
		t.Fatalf("expected trimmed user verification enum, got %q", got)
	}
	if got := conf.GetPasskeyAuthenticatorAttachment(); got != "platform" {
		t.Fatalf("expected trimmed authenticator attachment enum, got %q", got)
	}

	conf.Auth.Passkey.UserVerification = "bad-value"
	conf.Auth.Passkey.AuthenticatorAttachment = "bad-value"
	if got := conf.GetPasskeyUserVerification(); got != "preferred" {
		t.Fatalf("expected invalid user verification to default, got %q", got)
	}
	if got := conf.GetPasskeyAuthenticatorAttachment(); got != "any" {
		t.Fatalf("expected invalid authenticator attachment to default, got %q", got)
	}
}

func TestValidateStorageConfigRejectsIncompleteRemoteMode(t *testing.T) {
	tests := []struct {
		name      string
		config    SystemConfig
		wantError string
	}{
		{
			name: "s3 missing bucket",
			config: SystemConfig{
				Common: commonState{
					StorageMode: "s3",
					S3: s3StorageState{
						Region:    "us-east-1",
						AccessKey: "access-key",
						SecretKey: "secret-key",
					},
				},
			},
			wantError: "s3 storage is not fully configured",
		},
		{
			name: "r2 missing account id",
			config: SystemConfig{
				Common: commonState{
					StorageMode: "r2",
					R2: r2StorageState{
						Bucket:    "bucket",
						AccessKey: "access-key",
						SecretKey: "secret-key",
					},
				},
			},
			wantError: "r2 storage is not fully configured",
		},
		{
			name: "local ignores remote drafts",
			config: SystemConfig{
				Common: commonState{
					StorageMode: "local",
					S3: s3StorageState{
						PublicBaseURL: "https://cdn.example.com",
					},
				},
			},
		},
		{
			name: "complete r2",
			config: SystemConfig{
				Common: commonState{
					StorageMode: "r2",
					R2: r2StorageState{
						AccountID: "account-id",
						Bucket:    "bucket",
						AccessKey: "access-key",
						SecretKey: "secret-key",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateStorageConfig()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("expected storage config to pass, got %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("expected error %q, got %v", tt.wantError, err)
			}
		})
	}
}

func TestUpdateConfigRejectsIncompleteRemoteStorageBeforeMutating(t *testing.T) {
	current := &SystemConfig{
		Common: commonState{
			StorageMode: "local",
		},
	}
	next := &SystemConfig{
		Common: commonState{
			StorageMode: "s3",
			S3: s3StorageState{
				Region:    "us-east-1",
				AccessKey: "access-key",
				SecretKey: "secret-key",
			},
		},
	}

	err := current.UpdateConfig(next)
	if err == nil || err.Error() != "s3 storage is not fully configured" {
		t.Fatalf("expected incomplete s3 config error, got %v", err)
	}
	if current.GetStorageMode() != "local" {
		t.Fatalf("expected existing storage mode to remain local, got %q", current.GetStorageMode())
	}
}

func TestUpdateConfigKeepsRuntimeStateWhenSaveFails(t *testing.T) {
	current := &SystemConfig{
		General: generalState{
			Backend:  "https://old.example.com",
			TimeZone: "UTC",
		},
		Common: commonState{
			StorageMode: "local",
		},
	}
	next := &SystemConfig{
		General: generalState{
			Backend:  "https://new.example.com",
			TimeZone: "Asia/Tokyo",
		},
		Common: commonState{
			StorageMode: "local",
		},
	}

	previousSave := saveSystemConfig
	saveSystemConfig = func(*SystemConfig) error {
		return errors.New("simulated save failure")
	}
	t.Cleanup(func() {
		saveSystemConfig = previousSave
	})

	err := current.UpdateConfig(next)
	if err == nil || err.Error() != "simulated save failure" {
		t.Fatalf("expected simulated save failure, got %v", err)
	}
	if current.GetBackend() != "https://old.example.com" {
		t.Fatalf("expected runtime backend to remain unchanged, got %q", current.GetBackend())
	}
	if current.GetTimeZone() != "UTC" {
		t.Fatalf("expected runtime timezone to remain unchanged, got %q", current.GetTimeZone())
	}
}
